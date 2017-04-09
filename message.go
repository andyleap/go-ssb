package ssb

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type SignedMessage struct {
	Message
	Signature Signature `json:"signature"`
}

type Message struct {
	Previous  *Ref            `json:"previous"`
	Author    Ref             `json:"author"`
	Sequence  int             `json:"sequence"`
	Timestamp int             `json:"timestamp"`
	Hash      string          `json:"hash"`
	Content   json.RawMessage `json:"content"`
}

func Encode(i interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(i)
	if err != nil {
		return nil, err
	}
	return bytes.Trim(buf.Bytes(), "\n"), nil
}

func (m *SignedMessage) Verify(f *Feed) error {
	buf, err := Encode(m.Message)
	if err != nil {
		return err
	}
	err = m.Signature.Verify(buf, m.Author)
	if err != nil {
		return err
	}
	latest := f.Latest()
	if latest == nil && m.Sequence == 1 {
		return nil
	}
	if m.Previous == nil && latest == nil {
		return nil
	}
	if m.Previous == nil && latest != nil {
		return fmt.Errorf("Error: expected previous %s but found %s", latest.Key(), "")
	}
	if *m.Previous != latest.Key() {
		return fmt.Errorf("Error: expected previous %s but found %s", latest.Key(), *m.Previous)
	}
	if m.Sequence != latest.Sequence+1 || m.Timestamp <= latest.Timestamp {
		return fmt.Errorf("Error: out of order")
	}
	return nil
}

func (m *SignedMessage) Encode() string {
	buf, _ := Encode(m)
	return string(buf)
}

func (m *SignedMessage) Key() Ref {
	if m == nil {
		return ""
	}
	buf, _ := Encode(m)
	switch strings.ToLower(m.Hash) {
	case "sha256":
		hash := sha256.Sum256(buf)
		return Ref("%" + base64.StdEncoding.EncodeToString(hash[:]) + ".sha256")
	}
	return ""
}

func (m *Message) Sign(k Key) *SignedMessage {
	content, _ := Encode(m)
	sig := base64.StdEncoding.EncodeToString(k.Sign(content)) + ".sig." + k.Curve
	return &SignedMessage{Message: *m, Signature: Signature(sig)}
}
