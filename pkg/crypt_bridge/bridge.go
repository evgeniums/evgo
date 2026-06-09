// Package crypt_bridge loads libhatnfileencryptor at runtime via purego (no cgo)
// and exposes Encrypt and Decrypt functions wrapping hatn CryptContainer.
//
// The shared library is located via:
//  1. Env var WHITEM_FILE_ENCRYPTOR_LIB (absolute path, takes precedence).
//  2. <exe-dir>/native/libhatnfileencryptor.<ext>  (ext = dylib on darwin, so on linux).
//
// Contexts (hatn App + cipher suites) are created lazily per (configFile, sectionPath)
// pair and cached for the process lifetime.
//
// Concurrency: a package-level mutex serialises calls to each C function.
// These are low-frequency admin operations, so a global lock is acceptable and
// eliminates any OpenSSL/plugin reentrancy risk.
package crypt_bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// --------------------------------------------------------------------------
// C struct layouts
// --------------------------------------------------------------------------

// hatnFileEncryptorBytes mirrors hatn_file_encryptor_bytes.
type hatnFileEncryptorBytes struct {
	Data   uintptr
	Length uint64
}

// hatnFileDecryptorBytes mirrors hatn_file_decryptor_bytes.
type hatnFileDecryptorBytes struct {
	Data   uintptr
	Length uint64
}

// --------------------------------------------------------------------------
// Bound C function variables
// --------------------------------------------------------------------------

var (
	// encryptor
	fnCtxCreate  func(cfgFile *byte, cfgFileLen uint64, secPath *byte, secPathLen uint64, outCtx *uintptr) int32
	fnEncrypt    func(ctx uintptr, pt *byte, ptLen uint64, pass *byte, passLen uint64, out *hatnFileEncryptorBytes) int32
	fnBytesFree  func(buf *hatnFileEncryptorBytes)
	fnCtxDestroy func(ctx uintptr)
	fnLastError  func() uintptr

	// decryptor
	fnDecCtxCreate  func(cfgFile *byte, cfgFileLen uint64, secPath *byte, secPathLen uint64, outCtx *uintptr) int32
	fnDecrypt       func(ctx uintptr, ct *byte, ctLen uint64, pass *byte, passLen uint64, out *hatnFileDecryptorBytes) int32
	fnDecBytesFree  func(buf *hatnFileDecryptorBytes)
	fnDecCtxDestroy func(ctx uintptr)
	fnDecLastError  func() uintptr
)

// --------------------------------------------------------------------------
// Library loading (once per process)
// --------------------------------------------------------------------------

var (
	loadOnce sync.Once
	loadErr  error
)

// Load explicitly loads the shared library.  Call this at server startup to
// detect a missing or broken library early rather than on the first call.
// Safe to call multiple times; the library is loaded at most once.
func Load() error {
	return ensureLoaded()
}

func ensureLoaded() error {
	loadOnce.Do(func() {
		path, err := resolveLibPath()
		if err != nil {
			loadErr = err
			return
		}
		handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			loadErr = fmt.Errorf("crypt_bridge: dlopen %s: %w", path, err)
			return
		}

		// encryptor symbols
		purego.RegisterLibFunc(&fnCtxCreate, handle, "hatn_file_encryptor_ctx_create")
		purego.RegisterLibFunc(&fnEncrypt, handle, "hatn_file_encryptor_encrypt")
		purego.RegisterLibFunc(&fnBytesFree, handle, "hatn_file_encryptor_bytes_free")
		purego.RegisterLibFunc(&fnCtxDestroy, handle, "hatn_file_encryptor_ctx_destroy")
		purego.RegisterLibFunc(&fnLastError, handle, "hatn_file_encryptor_last_error")

		// decryptor symbols
		purego.RegisterLibFunc(&fnDecCtxCreate, handle, "hatn_file_decryptor_ctx_create")
		purego.RegisterLibFunc(&fnDecrypt, handle, "hatn_file_decryptor_decrypt")
		purego.RegisterLibFunc(&fnDecBytesFree, handle, "hatn_file_decryptor_bytes_free")
		purego.RegisterLibFunc(&fnDecCtxDestroy, handle, "hatn_file_decryptor_ctx_destroy")
		purego.RegisterLibFunc(&fnDecLastError, handle, "hatn_file_decryptor_last_error")
	})
	return loadErr
}

func resolveLibPath() (string, error) {
	if p := os.Getenv("WHITEM_FILE_ENCRYPTOR_LIB"); p != "" {
		return p, nil
	}
	ext := "so"
	if runtime.GOOS == "darwin" {
		ext = "dylib"
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("crypt_bridge: cannot determine executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), "native", "libhatnfileencryptor."+ext), nil
}

// --------------------------------------------------------------------------
// Context caches — encryptor and decryptor are independent
// --------------------------------------------------------------------------

type ctxKey struct{ configFile, sectionPath string }

var (
	// encryptor context cache
	encMu      sync.Mutex
	encCtxMap  = map[ctxKey]uintptr{}
	encCtxOnce = map[ctxKey]*sync.Once{}
	encCtxErr  = map[ctxKey]error{}

	// decryptor context cache
	decMu      sync.Mutex
	decCtxMap  = map[ctxKey]uintptr{}
	decCtxOnce = map[ctxKey]*sync.Once{}
	decCtxErr  = map[ctxKey]error{}
)

func getEncryptorContext(configFile, sectionPath string) (uintptr, error) {
	return getCtx(configFile, sectionPath,
		&encMu, encCtxOnce, encCtxMap, encCtxErr,
		func(cfgB, secB []byte) (uintptr, int32) {
			var ctx uintptr
			rc := fnCtxCreate(
				bytesPtr(cfgB), uint64(len(cfgB)),
				bytesPtr(secB), uint64(len(secB)),
				&ctx,
			)
			runtime.KeepAlive(cfgB)
			runtime.KeepAlive(secB)
			return ctx, rc
		},
		func() string { return cStrToGo(fnLastError()) },
	)
}

func getDecryptorContext(configFile, sectionPath string) (uintptr, error) {
	return getCtx(configFile, sectionPath,
		&decMu, decCtxOnce, decCtxMap, decCtxErr,
		func(cfgB, secB []byte) (uintptr, int32) {
			var ctx uintptr
			rc := fnDecCtxCreate(
				bytesPtr(cfgB), uint64(len(cfgB)),
				bytesPtr(secB), uint64(len(secB)),
				&ctx,
			)
			runtime.KeepAlive(cfgB)
			runtime.KeepAlive(secB)
			return ctx, rc
		},
		func() string { return cStrToGo(fnDecLastError()) },
	)
}

// getCtx is the generic lazy-create-and-cache logic used by both sides.
func getCtx(
	configFile, sectionPath string,
	mu *sync.Mutex,
	onceMap map[ctxKey]*sync.Once,
	ctxMap map[ctxKey]uintptr,
	errMap map[ctxKey]error,
	create func(cfgB, secB []byte) (uintptr, int32),
	lastErr func() string,
) (uintptr, error) {
	key := ctxKey{configFile, sectionPath}

	mu.Lock()
	once, ok := onceMap[key]
	if !ok {
		once = &sync.Once{}
		onceMap[key] = once
	}
	mu.Unlock()

	once.Do(func() {
		cfgB := stringToBytes(configFile)
		secB := stringToBytes(sectionPath)
		ctx, rc := create(cfgB, secB)

		mu.Lock()
		defer mu.Unlock()
		if rc != 0 {
			errMap[key] = fmt.Errorf("crypt_bridge: ctx_create failed (code %d): %s", rc, lastErr())
		} else {
			ctxMap[key] = ctx
		}
	})

	mu.Lock()
	defer mu.Unlock()
	if err := errMap[key]; err != nil {
		return 0, err
	}
	return ctxMap[key], nil
}

// --------------------------------------------------------------------------
// Public API
// --------------------------------------------------------------------------

// Encrypt encrypts plain using the default cipher suite from the hatn App config
// at (configFile, sectionPath) and the provided passphrase.
//
// The returned ciphertext is a CryptContainer blob byte-compatible with hatn's
// CryptContainer::unpack / parseAccountConfig on the C++ client side.
func Encrypt(configFile, sectionPath string, plain []byte, passphrase string) ([]byte, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	ctx, err := getEncryptorContext(configFile, sectionPath)
	if err != nil {
		return nil, err
	}

	encMu.Lock()
	defer encMu.Unlock()

	ptBytes := plain
	ppBytes := stringToBytes(passphrase)

	var out hatnFileEncryptorBytes
	rc := fnEncrypt(
		ctx,
		bytesPtr(ptBytes), uint64(len(ptBytes)),
		bytesPtr(ppBytes), uint64(len(ppBytes)),
		&out,
	)
	runtime.KeepAlive(ptBytes)
	runtime.KeepAlive(ppBytes)

	if rc != 0 {
		return nil, fmt.Errorf("crypt_bridge: encrypt failed (code %d): %s", rc, cStrToGo(fnLastError()))
	}
	result := encBytesToGo(out)
	fnBytesFree(&out)
	return result, nil
}

// Decrypt decrypts a CryptContainer blob produced by Encrypt (or
// hatn-file-encryptor) using the cipher suites from (configFile, sectionPath)
// and the provided passphrase.
func Decrypt(configFile, sectionPath string, ciphertext []byte, passphrase string) ([]byte, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	ctx, err := getDecryptorContext(configFile, sectionPath)
	if err != nil {
		return nil, err
	}

	decMu.Lock()
	defer decMu.Unlock()

	ctBytes := ciphertext
	ppBytes := stringToBytes(passphrase)

	var out hatnFileDecryptorBytes
	rc := fnDecrypt(
		ctx,
		bytesPtr(ctBytes), uint64(len(ctBytes)),
		bytesPtr(ppBytes), uint64(len(ppBytes)),
		&out,
	)
	runtime.KeepAlive(ctBytes)
	runtime.KeepAlive(ppBytes)

	if rc != 0 {
		return nil, fmt.Errorf("crypt_bridge: decrypt failed (code %d): %s", rc, cStrToGo(fnDecLastError()))
	}
	result := decBytesToGo(out)
	fnDecBytesFree(&out)
	return result, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func stringToBytes(s string) []byte {
	if len(s) == 0 {
		return []byte{0} // dummy non-nil pointer; length 0 passed separately
	}
	return []byte(s)
}

func bytesPtr(b []byte) *byte {
	return &b[0]
}

func encBytesToGo(b hatnFileEncryptorBytes) []byte {
	if b.Data == 0 || b.Length == 0 {
		return nil
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(b.Data)), b.Length)
	dst := make([]byte, b.Length)
	copy(dst, src)
	return dst
}

func decBytesToGo(b hatnFileDecryptorBytes) []byte {
	if b.Data == 0 || b.Length == 0 {
		return nil
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(b.Data)), b.Length)
	dst := make([]byte, b.Length)
	copy(dst, src)
	return dst
}

func cStrToGo(ptr uintptr) string {
	if ptr == 0 {
		return "(no error message)"
	}
	var n int
	for *(*byte)(unsafe.Pointer(ptr + uintptr(n))) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n))
}
