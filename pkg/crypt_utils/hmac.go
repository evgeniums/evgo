package crypt_utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"

	"github.com/evgeniums/evgo/pkg/utils"
)

type Hmac struct {
	Hash         hash.Hash
	StringCoding utils.StringCoding
}

func (h *Hmac) Sum() []byte {
	return h.Hash.Sum(nil)
}

func (h *Hmac) Calc(data ...[]byte) []byte {
	for _, block := range data {
		if data != nil {
			h.Hash.Write(block)
		}
	}
	return h.Hash.Sum(nil)
}

func (h *Hmac) CalcStrings(data ...string) []byte {
	for _, block := range data {
		if block != "" {
			h.Hash.Write([]byte(block))
		}
	}
	return h.Hash.Sum(nil)
}

func (h *Hmac) CalcStringsStr(data ...string) string {
	for _, block := range data {
		if block != "" {
			h.Hash.Write([]byte(block))
		}
	}
	return h.StringCoding.Encode(h.Hash.Sum(nil))
}

func (h *Hmac) CalcStr(data []byte) string {
	return h.StringCoding.Encode(h.Calc(data))
}

func (h *Hmac) SumStr() string {
	return h.StringCoding.Encode(h.Hash.Sum(nil))
}

func (h *Hmac) CalcStrStr(data string) string {
	return h.CalcStr([]byte(data))
}

func (h *Hmac) Check(sum []byte) bool {
	return hmac.Equal(h.Sum(), sum)
}

func (h *Hmac) CheckStr(sum string) error {

	sumB, err := h.StringCoding.Decode(sum)
	if err != nil {
		return fmt.Errorf("failed to decode sum: %s", err)
	}

	if !hmac.Equal(h.Sum(), sumB) {
		return errors.New("invalid HMAC")
	}

	return nil
}

type DigestBuilder = func() hash.Hash

func NewHmac(secret string, digestBuilder ...DigestBuilder) *Hmac {
	var builder = utils.OptionalArg(sha256.New, digestBuilder...)
	hm := hmac.New(builder, []byte(secret))
	h := &Hmac{Hash: hm}
	h.StringCoding = &utils.Base64StringCoding{}
	return h
}

func NewHmacCoding(secret string, val utils.StringCoding, digestBuilder ...DigestBuilder) *Hmac {
	var builder = utils.OptionalArg(sha256.New, digestBuilder...)
	hm := hmac.New(builder, []byte(secret))
	h := &Hmac{Hash: hm}
	h.StringCoding = val
	return h
}
