package crypt_utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"

	"github.com/evgeniums/evgo/pkg/utils"
)

type RsaSigner struct {
	utils.WithStringCoderBase
	key *rsa.PrivateKey
}

func NewRsaSigner(encoder ...utils.StringCoding) *RsaSigner {
	r := &RsaSigner{}
	r.WithStringCoderBase.Construct(encoder...)
	return r
}

func (r *RsaSigner) LoadKeyFromFile(filePath string, password string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return errors.New("no RSA private key found")
	}
	return r.LoadKey(data, password)
}

func (r *RsaSigner) LoadKey(data []byte, password string) (err error) {

	privPem, _ := pem.Decode(data)
	var privPemBytes []byte
	if privPem.Type != "RSA PRIVATE KEY" {
		return errors.New("RSA private key is of the wrong type")
	}

	if password != "" {
		privPemBytes, err = x509.DecryptPEMBlock(privPem, []byte(password))
		if err != nil {
			return errors.New("unable to decrypt passphrase")
		}
	} else {
		privPemBytes = privPem.Bytes
	}

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err != nil {
			return errors.New("unable to parse RSA private key")
		}
	}

	var ok bool
	r.key, ok = parsedKey.(*rsa.PrivateKey)
	if !ok {
		return errors.New("unable to parse RSA private key")
	}

	return nil
}

func (r *RsaSigner) Sign(data []byte, extraData ...string) ([]byte, error) {

	hashed := H256(data, extraData...)

	signature, err := rsa.SignPKCS1v15(rand.Reader, r.key, crypto.SHA256, hashed)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func (r *RsaSigner) SignB64(data []byte, extraData ...string) (string, error) {

	signature, err := r.Sign(data, extraData...)
	if err != nil {
		return string(signature), err
	}

	return utils.Base64Encode(signature), nil
}

func (r *RsaSigner) Key() *rsa.PrivateKey {
	return r.key
}
