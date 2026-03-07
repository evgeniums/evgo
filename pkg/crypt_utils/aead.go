package crypt_utils

import (
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"errors"

	"github.com/evgeniums/evgo/pkg/utils"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/pbkdf2"
)

type Pbkdf2Config struct {
	Iter        int
	HashBuilder DigestBuilder
}

func DefaultPbkdfConfig() Pbkdf2Config {
	return Pbkdf2Config{256, sha256.New}
}

type PbkdfFnc = func(password []byte, salt []byte, keySize int) []byte

type PbkdfConfig struct {
	Iter        int
	HashBuilder func() DigestBuilder
}

type MakeAeadFnc = func(key []byte) (cipher.AEAD, error)

type AEADConfig struct {
	MakeAead  MakeAeadFnc
	DeriveKey PbkdfFnc
	KeySize   int
}

func DefaultAEADConfig(pbkdfCfg Pbkdf2Config) AEADConfig {
	cfg := AEADConfig{}
	cfg.MakeAead = chacha20poly1305.NewX
	cfg.KeySize = chacha20poly1305.KeySize
	cfg.DeriveKey = func(password []byte, salt []byte, keySize int) []byte {
		return pbkdf2.Key(password, salt, pbkdfCfg.Iter, keySize, pbkdfCfg.HashBuilder)
	}
	return cfg
}

type AEAD struct {
	Cipher cipher.AEAD
}

func NewAEAD(secret string, salt []byte, config ...AEADConfig) (*AEAD, error) {
	var err error
	a := &AEAD{}

	var cfg AEADConfig
	if len(config) == 1 {
		cfg = config[0]
	} else {
		cfg = DefaultAEADConfig(DefaultPbkdfConfig())
	}

	key := cfg.DeriveKey([]byte(secret), salt, cfg.KeySize)
	a.Cipher, err = cfg.MakeAead(key)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *AEAD) Encrypt(plaintext []byte, additionalData ...[]byte) ([]byte, error) {

	// prepare buffer for nonce and ciphertext
	nonce := make([]byte, a.Cipher.NonceSize(), a.Cipher.NonceSize()+len(plaintext)+a.Cipher.Overhead())
	if _, err := cryptorand.Read(nonce); err != nil {
		return nil, err
	}

	// encrypt and seal data
	extraData := utils.OptionalArg(nil, additionalData...)
	ciphertext := a.Cipher.Seal(nonce, nonce, plaintext, extraData)

	// done
	return ciphertext, nil
}

func (a *AEAD) Decrypt(ciphertext []byte, additionalData ...[]byte) ([]byte, error) {

	// check length
	if len(ciphertext) < a.Cipher.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	// split nonce and ciphertext
	nonce := ciphertext[:a.Cipher.NonceSize()]
	ciphertextSplit := ciphertext[a.Cipher.NonceSize():]

	// decrypt the message and check it wasn't tampered with
	extraData := utils.OptionalArg(nil, additionalData...)
	plaintext, err := a.Cipher.Open(nil, nonce, ciphertextSplit, extraData)
	if err != nil {
		return nil, err
	}

	// done
	return plaintext, nil
}

func Decrypt(secret string, salt []byte, ciphertext []byte, additionalData ...[]byte) ([]byte, error) {

	aead, err := NewAEAD(secret, salt)
	if err != nil {
		return nil, err
	}

	return aead.Decrypt(ciphertext, additionalData...)
}

func DecryptStrings(secret string, salt string, ciphertext string, additionalStrings ...string) ([]byte, error) {

	b64 := utils.Base64StringCoding{}
	cipherdata, err := b64.Decode(ciphertext)
	if err != nil {
		return nil, err
	}
	additionalData := make([][]byte, 0, len(additionalStrings))
	for _, str := range additionalStrings {
		data, err := b64.Decode(str)
		if err != nil {
			return nil, err
		}
		additionalData = append(additionalData, data)
	}

	return Decrypt(secret, []byte(salt), cipherdata, additionalData...)
}

func AeadEncryptB64(a *AEAD, plaintext []byte, additionalData ...[]byte) (string, error) {
	b, err := a.Encrypt(plaintext, additionalData...)
	if err != nil {
		return "", err
	}
	c := utils.Base64StringCoding{}
	return c.Encode(b), nil
}
