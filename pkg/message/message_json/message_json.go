package message_json

import (
	"encoding/json"
)

// JSON serializer.
type JsonSerializer struct {
}

func (j *JsonSerializer) ParseMessage(data []byte, message any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, message)
}

func (j *JsonSerializer) SerializeMessage(message any) ([]byte, error) {
	return json.Marshal(message)
}

func (j *JsonSerializer) Format() string {
	return "json"
}

func (j *JsonSerializer) ContentMime() string {
	return "application/json"
}

var Serializer = &JsonSerializer{}
