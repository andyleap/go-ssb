package ssb

import (
	"encoding/json"
)

type MessageBody struct {
	Type    string   `json:"type"`
	Message *Message `json:"-"`
}

var MessageTypes = map[string]func(mb MessageBody) interface{}{}

func (m *Message) DecodeMessage() (t string, mb interface{}) {
	Type := &MessageBody{}
	json.Unmarshal(m.Content, &Type)
	Type.Message = m
	if mf, ok := MessageTypes[Type.Type]; ok {
		mb = mf(*Type)
	}
	t = Type.Type
	json.Unmarshal(m.Content, &mb)
	return
}
