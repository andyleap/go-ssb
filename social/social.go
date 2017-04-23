package social

import (
	"encoding/json"

	"github.com/andyleap/go-ssb"
	"github.com/boltdb/bolt"
)

type Link struct {
	Link ssb.Ref `json:"link"`
}

type Post struct {
	ssb.MessageBody
	Text     string  `json:"text"`
	Channel  string  `json:"channel,omitempty"`
	Root     ssb.Ref `json:"root,omitempty"`
	Branch   ssb.Ref `json:"branch,omitempty"`
	Recps    []Link  `json:"recps,omitempty"`
	Mentions []Link  `json:"mentions,omitempty"`
}

type About struct {
	ssb.MessageBody
	About ssb.Ref `json:"about"`
	Name  string  `json:"name,omitempty"`
	Image ssb.Ref `json:"image,omitempty"`
}

type Vote struct {
	ssb.MessageBody
	Vote struct {
		Link   ssb.Ref `json:"link"`
		Value  int     `json:"value"`
		Reason string  `json:"reason,omitempty"`
	} `json:"vote"`
}

func init() {
	ssb.MessageTypes["post"] = func() interface{} { return &Post{} }
	ssb.MessageTypes["about"] = func() interface{} { return &About{} }
	ssb.MessageTypes["vote"] = func() interface{} { return &Vote{} }
	ssb.AddMessageHooks["social"] = func(m *ssb.SignedMessage, tx *bolt.Tx) error {
		_, mb := m.DecodeMessage()
		if mba, ok := mb.(*About); ok {
			if mba.About == m.Author {
				FeedsBucket, err := tx.CreateBucketIfNotExists([]byte("feeds"))
				if err != nil {
					return err
				}
				FeedBucket, err := FeedsBucket.CreateBucketIfNotExists(m.Author.DBKey())
				if err != nil {
					return err
				}
				aboutdata := FeedBucket.Get([]byte("about"))
				var a About
				if aboutdata != nil {
					json.Unmarshal(aboutdata, &a)
				}
				if mba.Name != "" {
					a.Name = mba.Name
				}
				if mba.Image.Type != ssb.RefInvalid {
					a.Image = mba.Image
				}
				buf, err := json.Marshal(a)
				if err != nil {
					return err
				}
				err = FeedBucket.Put([]byte("about"), buf)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func GetAbout(tx *bolt.Tx, ref ssb.Ref) (a *About) {
	FeedsBucket := tx.Bucket([]byte("feeds"))
	if FeedsBucket == nil {
		return
	}
	FeedBucket := FeedsBucket.Bucket(ref.DBKey())
	if FeedBucket == nil {
		return
	}
	aboutdata := FeedBucket.Get([]byte("about"))
	if aboutdata == nil {
		return
	}
	json.Unmarshal(aboutdata, &a)
	return
}
