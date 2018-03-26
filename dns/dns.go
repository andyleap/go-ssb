package dns

import (
	"encoding/json"

	"github.com/boltdb/bolt"

	"github.com/andyleap/go-ssb"
)

type Record struct {
	Name  string          `json:"name"`
	Class string          `json:"class"`
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data"`
}

type DNS struct {
	ssb.MessageBody
	Record Record    `json:"record"`
	Branch []ssb.Ref `json:"branch"`
}

func init() {
	ssb.MessageTypes["ssb-dns"] = func(_ ssb.MessageBody) interface{} { return &DNS{} }
	ssb.RebuildClearHooks["dns"] = func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte("dns"))
	}
	ssb.AddMessageHooks["dns"] = func(m *ssb.SignedMessage, tx *bolt.Tx) error {
		_, mb := m.DecodeMessage()
		if mbr, ok := mb.(*DNS); ok {
			PubBucket, err := tx.CreateBucketIfNotExists([]byte("dns"))
			if err != nil {
				return err
			}
			buf, _ := json.Marshal(mbr)
			err = PubBucket.Put(m.Key().DBKey(), buf)
			if err != nil {
				return err
			}
			return nil
		}
		return nil
	}
}
