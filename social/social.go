package social

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/andyleap/go-ssb"
	"github.com/boltdb/bolt"
)

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

type Link struct {
	Link ssb.Ref `json:"link"`
}

type Image struct {
	image
}

type image struct {
	Link   ssb.Ref `json:"link"`
	Width  int     `json:"width,omitempty"`
	Height int     `json:"height,omitempty"`
	Name   string  `json:"name,omitempty"`
	Size   int     `json:"size,omitempty"`
	Type   string  `json:"type,omitempty"`
}

func (i *Image) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &i.image)
	if err != nil {
		err = json.Unmarshal(b, &i.Link)
		if err != nil {
			return err
		}
	}
	return nil
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
	Image *Image  `json:"image,omitempty"`
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
	ssb.MessageTypes["post"] = func(mb ssb.MessageBody) interface{} { return &Post{MessageBody: mb} }
	ssb.MessageTypes["about"] = func(mb ssb.MessageBody) interface{} { return &About{MessageBody: mb} }
	ssb.MessageTypes["vote"] = func(mb ssb.MessageBody) interface{} { return &Vote{MessageBody: mb} }
	ssb.RebuildClearHooks["social"] = func(tx *bolt.Tx) error {
		tx.DeleteBucket([]byte("votes"))
		tx.DeleteBucket([]byte("threads"))
		b, _ := tx.CreateBucketIfNotExists([]byte("feeds"))
		b.ForEach(func(k, v []byte) error {
			b.Bucket(k).Delete([]byte("about"))
			return nil
		})

		return nil
	}
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
				if mba.Image != nil {
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
		if vote, ok := mb.(*Vote); ok {
			VotesBucket, err := tx.CreateBucketIfNotExists([]byte("votes"))
			if err != nil {
				return err
			}
			votesRaw := VotesBucket.Get(vote.Vote.Link.DBKey())
			var votes []ssb.Ref
			if votesRaw != nil {
				json.Unmarshal(votesRaw, &votes)
			}
			votes = append(votes, m.Key())
			buf, _ := json.Marshal(votes)

			err = VotesBucket.Put(vote.Vote.Link.DBKey(), buf)
			if err != nil {
				return err
			}
		}
		if post, ok := mb.(*Post); ok {
			if post.Root.Type != ssb.RefInvalid {
				ThreadsBucket, err := tx.CreateBucketIfNotExists([]byte("threads"))
				if err != nil {
					return err
				}
				ThreadBucket, err := ThreadsBucket.CreateBucketIfNotExists(post.Root.DBKey())
				if err != nil {
					return err
				}
				logBucket, err := ThreadBucket.CreateBucketIfNotExists([]byte("log"))
				if err != nil {
					return err
				}
				logBucket.FillPercent = 1
				seq, err := logBucket.NextSequence()
				if err != nil {
					return err
				}
				logBucket.Put(itob(int(seq)), m.Key().DBKey())

				timeBucket, err := ThreadBucket.CreateBucketIfNotExists([]byte("time"))
				if err != nil {
					return err
				}
				i := int(m.Timestamp * float64(time.Millisecond))
				for timeBucket.Get(itob(i)) != nil {
					i++
				}
				timeBucket.Put(itob(i), m.Key().DBKey())
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

func GetVotes(tx *bolt.Tx, ref ssb.Ref) []*ssb.SignedMessage {
	VotesBucket := tx.Bucket([]byte("votes"))
	if VotesBucket == nil {
		return nil
	}
	votesRaw := VotesBucket.Get(ref.DBKey())
	var voteRefs []ssb.Ref
	if votesRaw != nil {
		json.Unmarshal(votesRaw, &voteRefs)
	}
	votes := make([]*ssb.SignedMessage, 0, len(voteRefs))
	for _, r := range voteRefs {
		msg := ssb.GetMsg(tx, r)
		if msg == nil {
			continue
		}
		votes = append(votes, msg)
	}
	return votes
}

func GetThread(tx *bolt.Tx, ref ssb.Ref) []*ssb.SignedMessage {
	ThreadsBucket := tx.Bucket([]byte("threads"))
	if ThreadsBucket == nil {
		return nil
	}
	ThreadBucket := ThreadsBucket.Bucket(ref.DBKey())
	if ThreadBucket == nil {
		return nil
	}
	timeBucket := ThreadBucket.Bucket([]byte("time"))
	if timeBucket == nil {
		return nil
	}
	thread := []*ssb.SignedMessage{}
	timeBucket.ForEach(func(k, v []byte) error {
		msg := ssb.GetMsg(tx, ssb.DBRef(v))
		if msg != nil {
			thread = append(thread, msg)
		}
		return nil
	})
	return thread
}
