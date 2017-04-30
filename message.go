package ssb

import (
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	Timestamp float64         `json:"timestamp"`
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
	if latest == nil && m.Previous != nil {
		fmt.Println(string(m.Encode()))
		return fmt.Errorf("Expected message")
	}
	if m.Previous == nil && latest == nil {
		return nil
	}
	if m.Previous == nil && latest != nil {
		fmt.Println(string(m.Encode()))
		return fmt.Errorf("Error: expected previous %s but found %s", latest.Key(), "")
	}
	if latest != nil && m.Sequence == latest.Sequence {
		return fmt.Errorf("Error: Repeated message")
	}
	if *m.Previous != latest.Key() {
		/*buf, _ := Encode(latest)
		buf2 := ToJSBinary(buf)

		buf3, _ := Encode(m)
		fmt.Printf("\n%q\n%q\n%q\n", string(buf), string(buf2), string(buf3))*/
		return fmt.Errorf("Error: expected previous %s but found %s", latest.Key(), *m.Previous)
	}
	if m.Sequence != latest.Sequence+1 || m.Timestamp <= latest.Timestamp {
		return fmt.Errorf("Error: out of order")
	}
	return nil
}

func (m *SignedMessage) Encode() []byte {
	buf, _ := Encode(m)
	return buf
}

func (m *SignedMessage) Compress() []byte {
	buf := m.Encode()
	cbuf := bytes.Buffer{}
	cbuf.WriteByte(2)
	cwrite, _ := flate.NewWriterDict(&cbuf, 9, Compression2)
	cwrite.Write(buf)
	cwrite.Flush()
	return cbuf.Bytes()
}

func DecompressMessage(cbuf []byte) *SignedMessage {
	switch cbuf[0] {
	case 1:
		reader := flate.NewReaderDict(bytes.NewReader(cbuf[1:]), Compression1)
		buf, _ := ioutil.ReadAll(reader)
		reader.Close()
		var m *SignedMessage
		json.Unmarshal(buf, &m)
		return m
	case 2:
		reader := flate.NewReaderDict(bytes.NewReader(cbuf[1:]), Compression2)
		buf, _ := ioutil.ReadAll(reader)
		reader.Close()
		var m *SignedMessage
		json.Unmarshal(buf, &m)
		return m
	default:
		var m *SignedMessage
		json.Unmarshal(cbuf, &m)
		return m
	}

}

func (m *SignedMessage) Key() Ref {
	if m == nil {
		return Ref{}
	}
	buf, _ := Encode(m)
	/*enc := RemoveUnsupported(charmap.ISO8859_1.NewEncoder())
	buf, err := enc.Bytes(buf)
	if err != nil {
		panic(err)
	}*/
	buf = ToJSBinary(buf)
	switch strings.ToLower(m.Hash) {
	case "sha256":
		hash := sha256.Sum256(buf)
		ref, _ := NewRef(RefMessage, hash[:], RefAlgoSha256)
		return ref
	}
	fmt.Println(string(buf))
	return Ref{}
}

func (m *Message) Sign(s Signer) *SignedMessage {
	content, _ := Encode(m)
	sig := s.Sign(content)
	return &SignedMessage{Message: *m, Signature: sig}
}
