package http_request

import (
	"github.com/evgeniums/evgo/pkg/message"
)

type UrlEncodedSerializer struct {
	message.SerializerBase
}

func (s *UrlEncodedSerializer) SerializeMessage(message interface{}) ([]byte, error) {

	q, err := UrlEncode(message)
	if err != nil {
		return nil, err
	}

	return []byte(q), nil
}

func (s *UrlEncodedSerializer) Format() string {
	return "urlencoded"
}

func (s *UrlEncodedSerializer) ContentMime() string {
	return "application/x-www-form-urlencoded"
}
