package ssb

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/cryptix/secretstream/secrethandshake"
	"golang.org/x/crypto/ed25519"
)

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func btoi(b []byte) int {
	return int(binary.BigEndian.Uint64(b))
}

type DataStore struct {
	db *bolt.DB

	feedlock sync.Mutex
	feeds    map[Ref]*Feed

	Topic *MessageTopic

	PrimaryKey *secrethandshake.EdKeyPair
	PrimaryRef Ref

	extraData     map[string]interface{}
	extraDataLock sync.Mutex

	Keys map[Ref]Signer
}

func (ds *DataStore) ExtraData(name string) interface{} {
	ds.extraDataLock.Lock()
	defer ds.extraDataLock.Unlock()
	return ds.extraData[name]
}

func (ds *DataStore) SetExtraData(name string, data interface{}) {
	ds.extraDataLock.Lock()
	defer ds.extraDataLock.Unlock()
	ds.extraData[name] = data
}

func (ds *DataStore) DB() *bolt.DB {
	return ds.db
}

type Feed struct {
	store *DataStore
	ID    Ref

	Topic *MessageTopic
}

func OpenDataStore(path string, primaryKey string) (*DataStore, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	ds := &DataStore{
		db:        db,
		feeds:     map[Ref]*Feed{},
		Topic:     NewMessageTopic(),
		extraData: map[string]interface{}{},
		Keys:      map[Ref]Signer{},
	}
	ds.PrimaryKey, _ = secrethandshake.LoadSSBKeyPair(primaryKey)
	ds.PrimaryRef = Ref("@" + base64.StdEncoding.EncodeToString(ds.PrimaryKey.Public[:]) + ".ed25519")
	ds.Keys[Ref("@"+base64.StdEncoding.EncodeToString(ds.PrimaryKey.Public[:])+".ed25519")] = &SignerEd25519{ed25519.PrivateKey(ds.PrimaryKey.Secret[:])}
	return ds, nil
}

func (ds *DataStore) GetFeed(feedID Ref) *Feed {
	ds.feedlock.Lock()
	defer ds.feedlock.Unlock()
	if feed, ok := ds.feeds[feedID]; ok {
		return feed
	}
	if feedID.Type() != RefFeed {
		return nil
	}
	feed := &Feed{store: ds, ID: feedID, Topic: NewMessageTopic()}
	feed.Topic.Register(ds.Topic.Send, true)
	ds.feeds[feedID] = feed
	return feed
}

var AddMessageHooks = map[string]func(m *SignedMessage, tx *bolt.Tx) error{}

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
		LogBucket, err := tx.CreateBucketIfNotExists([]byte("log"))
		if err != nil {
			return err
		}
		seq, err := LogBucket.NextSequence()
		if err != nil {
			return err
		}
		LogBucket.Put(itob(int(seq)), []byte(m.Key()))
		for _, hook := range AddMessageHooks {
			hook(m, tx)
		}

		return nil
	})
	if err != nil {
		return err
	}
	f.Topic.Send <- m
	return nil
}

func (f *Feed) PublishMessage(body interface{}) error {
	content, _ := json.Marshal(body)

	m := &Message{
		Author:    f.ID,
		Timestamp: float64(time.Now().UnixNano() / int64(time.Millisecond)),
		Hash:      "sha256",
		Content:   content,
		Sequence:  1,
	}

	if l := f.Latest(); l != nil {
		key := l.Key()
		m.Previous = &key
		m.Sequence = l.Sequence + 1
		for m.Timestamp <= l.Timestamp {
			m.Timestamp += 0.01
		}
	}

	signer := f.store.Keys[f.ID]
	if signer == nil {
		return fmt.Errorf("Cannot sign message without signing key for feed")
	}
	sm := m.Sign(signer)

	err := f.AddMessage(sm)
	if err != nil {
		return err
	}

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

func (f *Feed) Log(seq int, live bool) chan *SignedMessage {
	c := make(chan *SignedMessage, 10)
	go func() {
		liveChan := make(chan *SignedMessage, 10)
		if live {
			f.Topic.Register(liveChan, false)
		} else {
			close(liveChan)
		}
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
				if m.Sequence < seq {
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
			if m.Sequence < seq {
				continue
			}
			seq = m.Sequence
			c <- m
		}
		close(c)
	}()
	return c
}
