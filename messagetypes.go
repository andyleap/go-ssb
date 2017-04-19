package ssb

import (
	"encoding/json"
)

type MessageBody struct {
	Type string `json:"type"`
}

var MessageTypes = map[string]func() interface{}{}

type Link struct {
	Link Ref `json:"link"`
}

type Post struct {
	MessageBody
	Text     string `json:"text"`
	Channel  string `json:"channel,omitempty"`
	Root     Ref    `json:"root,omitempty"`
	Branch   Ref    `json:"branch,omitempty"`
	Recps    []Link `json:"recps,omitempty"`
	Mentions []Link `json:"mentions,omitempty"`
}

type About struct {
	MessageBody
	About Ref    `json:"about"`
	Name  string `json:"name,omitempty"`
	Image Ref    `json:"image,omitempty"`
}

type Vote struct {
	MessageBody
	Vote struct {
		Link   Ref    `json:"link"`
		Value  int    `json:"value"`
		Reason string `json:"reason,omitempty"`
	} `json:"vote"`
}

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
