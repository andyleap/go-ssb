package ssb

import (
	"encoding/json"
)

type MessageBody struct {
	Type string `json:"type"`
}

var MessageTypes = map[string]func() interface{}{}

func (m *Message) DecodeMessage() (t string, mb interface{}) {
	Type := MessageBody{}
	json.Unmarshal(m.Content, &Type)
	if mf, ok := MessageTypes[Type.Type]; ok {
		mb = mf()
	}
	t = Type.Type
	json.Unmarshal(m.Content, &mb)
	return
}
