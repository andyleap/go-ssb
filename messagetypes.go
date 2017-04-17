package ssb

import (
	"encoding/json"
)

type MessageBody struct {
	Type string `json:"type"`
}

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

type Contact struct {
	MessageBody
	Contact   Ref   `json:"contact"`
	Following *bool `json:"following,omitempty"`
	Blocking  *bool `json:"blocking,omitempty"`
}

type Vote struct {
	MessageBody
	Vote struct {
		Link   Ref    `json:"link"`
		Value  int    `json:"value"`
		Reason string `json:"reason,omitempty"`
	} `json:"vote"`
}

type PubData struct {
	Link Ref    `json:"link"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Pub struct {
	MessageBody
	Pub PubData `json:"pub"`
}

func (m *Message) DecodeMessage() (t string, mb interface{}) {
	Type := MessageBody{}
	json.Unmarshal(m.Content, &Type)
	switch Type.Type {
	case "post":
		mb = &Post{}
	case "about":
		mb = &About{}
	case "contact":
		mb = &Contact{}
	case "vote":
		mb = &Vote{}
	case "pub":
		mb = &Pub{}
	}
	t = Type.Type
	json.Unmarshal(m.Content, &mb)
	return
}
