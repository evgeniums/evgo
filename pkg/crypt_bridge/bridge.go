// Package crypt_bridge loads libhatnfileencryptor at runtime via purego (no cgo)
// and exposes a single Encrypt function used to produce CryptContainer-compatible
// ciphertext byte-compatible with hatn's parseAccountConfig / CryptContainer::unpack.
//
// The shared library is located via:
//  1. Env var WHITEM_FILE_ENCRYPTOR_LIB (absolute path, takes precedence).
//  2. <exe-dir>/native/libhatnfileencryptor.<ext>  (ext = dylib on darwin, so on linux).
//
// A single encryption context (hatn App + cipher suites) is created lazily per
// (configFile, configRoot) pair and cached for the process lifetime.
//
// Concurrency: a package-level mutex serialises calls to the C encrypt function.
// Account-config generation is a low-frequency admin operation, so a global lock
// is acceptable and eliminates any OpenSSL/plugin reentrancy risk.
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
// C struct layout matching hatn_file_encryptor_bytes
// --------------------------------------------------------------------------

// hatnFileEncryptorBytes mirrors the C struct:
//
//	typedef struct { const char* data; size_t length; } hatn_file_encryptor_bytes;
type hatnFileEncryptorBytes struct {
	Data   uintptr
	Length uint64
}

// --------------------------------------------------------------------------
// Bound C function types
// --------------------------------------------------------------------------

var (
	fnCtxCreate  func(cfgFile *byte, cfgFileLen uint64, cfgRoot *byte, cfgRootLen uint64, outCtx *uintptr) int32
	fnEncrypt    func(ctx uintptr, pt *byte, ptLen uint64, pass *byte, passLen uint64, outCipher *hatnFileEncryptorBytes) int32
	fnBytesFree  func(buf *hatnFileEncryptorBytes)
	fnCtxDestroy func(ctx uintptr)
	fnLastError  func() uintptr // returns *char (static TLS string in C)
)

// --------------------------------------------------------------------------
// Library loading (once per process)
// --------------------------------------------------------------------------

var (
	loadOnce sync.Once
	loadErr  error
)

// Load explicitly loads the shared library.  Call this at server startup to
// detect a missing or broken library early rather than on the first Encrypt
// call.  Safe to call multiple times; the library is loaded at most once.
// Returns the load error so the caller can decide whether to abort or log and
// degrade gracefully.
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
		purego.RegisterLibFunc(&fnCtxCreate, handle, "hatn_file_encryptor_ctx_create")
		purego.RegisterLibFunc(&fnEncrypt, handle, "hatn_file_encryptor_encrypt")
		purego.RegisterLibFunc(&fnBytesFree, handle, "hatn_file_encryptor_bytes_free")
		purego.RegisterLibFunc(&fnCtxDestroy, handle, "hatn_file_encryptor_ctx_destroy")
		purego.RegisterLibFunc(&fnLastError, handle, "hatn_file_encryptor_last_error")
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
// Context cache
// --------------------------------------------------------------------------

type ctxKey struct{ configFile, configRoot string }

var (
	mu      sync.Mutex
	ctxMap  = map[ctxKey]uintptr{}
	ctxOnce = map[ctxKey]*sync.Once{}
	ctxErr  = map[ctxKey]error{}
)

func getContext(configFile, configRoot string) (uintptr, error) {
	key := ctxKey{configFile, configRoot}

	mu.Lock()
	once, ok := ctxOnce[key]
	if !ok {
		once = &sync.Once{}
		ctxOnce[key] = once
	}
	mu.Unlock()

	once.Do(func() {
		cfgFileBytes := stringToBytes(configFile)
		cfgRootBytes := stringToBytes(configRoot)

		var ctx uintptr
		rc := fnCtxCreate(
			bytesPtr(cfgFileBytes), uint64(len(cfgFileBytes)),
			bytesPtr(cfgRootBytes), uint64(len(cfgRootBytes)),
			&ctx,
		)
		runtime.KeepAlive(cfgFileBytes)
		runtime.KeepAlive(cfgRootBytes)

		mu.Lock()
		defer mu.Unlock()
		if rc != 0 {
			ctxErr[key] = fmt.Errorf("crypt_bridge: ctx_create failed (code %d): %s", rc, cLastError())
		} else {
			ctxMap[key] = ctx
		}
	})

	mu.Lock()
	defer mu.Unlock()
	if err := ctxErr[key]; err != nil {
		return 0, err
	}
	return ctxMap[key], nil
}

// --------------------------------------------------------------------------
// Public API
// --------------------------------------------------------------------------

// Encrypt encrypts plain using the default cipher suite from the hatn App config
// (configFile, configRoot) and the provided passphrase.
//
// The returned ciphertext is a CryptContainer blob byte-compatible with hatn's
// CryptContainer::unpack / parseAccountConfig on the C++ client side.
func Encrypt(configFile, configRoot string, plain []byte, passphrase string) ([]byte, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}

	ctx, err := getContext(configFile, configRoot)
	if err != nil {
		return nil, err
	}

	mu.Lock()
	defer mu.Unlock()

	ptBytes := plain
	ppBytes := stringToBytes(passphrase)

	var outCipher hatnFileEncryptorBytes
	rc := fnEncrypt(
		ctx,
		bytesPtr(ptBytes), uint64(len(ptBytes)),
		bytesPtr(ppBytes), uint64(len(ppBytes)),
		&outCipher,
	)
	runtime.KeepAlive(ptBytes)
	runtime.KeepAlive(ppBytes)

	if rc != 0 {
		return nil, fmt.Errorf("crypt_bridge: encrypt failed (code %d): %s", rc, cLastError())
	}

	result := cBytesToGo(outCipher)
	fnBytesFree(&outCipher)
	return result, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// stringToBytes returns a non-nil []byte for any string, including empty.
func stringToBytes(s string) []byte {
	if len(s) == 0 {
		return []byte{0} // dummy non-nil pointer, length 0 passed separately
	}
	return []byte(s)
}

// bytesPtr returns a pointer to the first byte of b.
// b must be non-nil and non-empty (len >= 1).
func bytesPtr(b []byte) *byte {
	return &b[0]
}

// cBytesToGo copies the C buffer into a Go-owned []byte.
func cBytesToGo(b hatnFileEncryptorBytes) []byte {
	if b.Data == 0 || b.Length == 0 {
		return nil
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(b.Data)), b.Length)
	dst := make([]byte, b.Length)
	copy(dst, src)
	return dst
}

// cLastError returns the thread-local error message from the C library.
func cLastError() string {
	ptr := fnLastError()
	if ptr == 0 {
		return "(no error message)"
	}
	// Walk the C string.
	var n int
	for p := ptr; *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0; n++ {
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n))
}
