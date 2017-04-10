package ssb

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func btoi(b []byte) int {
	return int(binary.BigEndian.Uint64(b))
}

type FeedStore struct {
	db *bolt.DB

	feedlock sync.Mutex
	feeds    map[Ref]*Feed

	Topic *MessageTopic
}

type Feed struct {
	store *FeedStore
	ID    Ref

	Topic *MessageTopic
}

func OpenFeedStore(path string) (*FeedStore, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	fs := &FeedStore{db: db, feeds: map[Ref]*Feed{}, Topic: NewMessageTopic()}
	go fs.HandleGraph()
	return fs, nil
}

func (fs *FeedStore) GetFeed(feedID Ref) *Feed {
	fs.feedlock.Lock()
	defer fs.feedlock.Unlock()
	if feed, ok := fs.feeds[feedID]; ok {
		return feed
	}
	if feedID.Type() != RefFeed {
		return nil
	}
	feed := &Feed{store: fs, ID: feedID, Topic: NewMessageTopic()}
	feed.Topic.Register(fs.Topic.Send, true)
	fs.feeds[feedID] = feed
	return feed
}

func (f *Feed) AddMessage(m *SignedMessage) error {
	if m.Author != f.ID {
		return fmt.Errorf("Wrong feed")
	}
	err := m.Verify(f)
	if err != nil {
		return err
	}
	err = f.store.db.Update(func(tx *bolt.Tx) error {
		FeedsBucket, err := tx.CreateBucketIfNotExists([]byte("feeds"))
		if err != nil {
			return err
		}
		FeedBucket, err := FeedsBucket.CreateBucketIfNotExists([]byte(f.ID))
		if err != nil {
			return err
		}
		buf, err := Encode(m)
		if err != nil {
			return err
		}
		FeedBucket.Put(itob(m.Sequence), buf)
		return nil
	})
	if err != nil {
		return err
	}
	f.Topic.Send <- m
	return nil
}

func (f *Feed) Latest() (m *SignedMessage) {
	f.store.db.View(func(tx *bolt.Tx) error {
		FeedsBucket := tx.Bucket([]byte("feeds"))
		if FeedsBucket == nil {
			return nil
		}
		FeedBucket := FeedsBucket.Bucket([]byte(f.ID))
		if FeedBucket == nil {
			return nil
		}
		cur := FeedBucket.Cursor()
		_, val := cur.Last()
		json.Unmarshal(val, &m)
		return nil
	})
	return
}

var ErrLogClosed = errors.New("LogClosed")

func (f *Feed) Log(seq int) chan *SignedMessage {
	c := make(chan *SignedMessage, 10)
	go func() {
		liveChan := make(chan *SignedMessage, 10)
		f.Topic.Register(liveChan, false)
		err := f.store.db.View(func(tx *bolt.Tx) error {
			FeedsBucket := tx.Bucket([]byte("feeds"))
			if FeedsBucket == nil {
				return nil
			}
			FeedBucket := FeedsBucket.Bucket([]byte(f.ID))
			if FeedBucket == nil {
				return nil
			}
			err := FeedBucket.ForEach(func(k, v []byte) error {
				var m *SignedMessage
				json.Unmarshal(v, &m)
				if m.Sequence <= seq {
					return nil
				}
				seq = m.Sequence
				select {
				case c <- m:
				case <-time.After(100 * time.Millisecond):
					close(c)
					return ErrLogClosed
				}
				return nil
			})
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return
		}
		for m := range liveChan {
			if m.Sequence <= seq {
				continue
			}
			seq = m.Sequence
			c <- m
		}
	}()
	return c
}
