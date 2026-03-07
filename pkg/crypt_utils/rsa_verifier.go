package crypt_utils

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/evgeniums/evgo/pkg/utils"
)

const RSA_H256_SIGNATURE = "rsa_h256_signature"

type RsaVerifier struct {
	utils.WithStringCoderBase
	key *rsa.PublicKey
}

func NewRsaVerifier(encoder ...utils.StringCoding) *RsaVerifier {
	r := &RsaVerifier{}
	r.WithStringCoderBase.Construct(encoder...)
	return r
}

func (r *RsaVerifier) LoadKeyFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return errors.New("no RSA public key found")
	}
	return r.LoadKey(data)
}

func (r *RsaVerifier) LoadKey(data []byte) (err error) {

	pubPem, _ := pem.Decode([]byte(data))
	if pubPem == nil {
		return errors.New("RSA public key not in pem format")
	}
	if pubPem.Type != "RSA PUBLIC KEY" && pubPem.Type != "PUBLIC KEY" {
		return errors.New("RSA public key is of the wrong type")
	}

	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKIXPublicKey(pubPem.Bytes); err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to parse RSA public key: %v", err.Error()))
	}

	var ok bool
	if r.key, ok = parsedKey.(*rsa.PublicKey); !ok {
		return errors.New("unable to parse RSA public key")
	}

	return nil
}

func (r *RsaVerifier) Verify(data []byte, signature []byte, extraData ...string) error {

	hashed := H256(data, extraData...)

	err := rsa.VerifyPKCS1v15(r.key, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return err
	}

	return nil
}
