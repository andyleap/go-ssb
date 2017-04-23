package ssb

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

	addChan chan *SignedMessage
}

type Pointer struct {
	Author   Ref
	Sequence int
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
	ds.PrimaryRef, _ = NewRef(RefFeed, ds.PrimaryKey.Public[:], RefAlgoEd25519)
	ds.Keys[ds.PrimaryRef] = &SignerEd25519{ed25519.PrivateKey(ds.PrimaryKey.Secret[:])}
	return ds, nil
}

func (ds *DataStore) GetFeed(feedID Ref) *Feed {
	ds.feedlock.Lock()
	defer ds.feedlock.Unlock()
	if feed, ok := ds.feeds[feedID]; ok {
		return feed
	}
	if feedID.Type != RefFeed {
		return nil
	}
	feed := &Feed{store: ds, ID: feedID, Topic: NewMessageTopic(), addChan: make(chan *SignedMessage, 10)}
	go func() {
		for m := range feed.addChan {
			feed.addMessage(m)
		}
	}()
	feed.Topic.Register(ds.Topic.Send, true)
	ds.feeds[feedID] = feed
	return feed
}

func (ds *DataStore) Get(tx *bolt.Tx, post Ref) (m *SignedMessage) {
	var err error
	if tx == nil {
		tx, err = ds.db.Begin(false)
		if err != nil {
			return
		}
		defer tx.Rollback()
	}
	PointerBucket := tx.Bucket([]byte("pointer"))
	if PointerBucket == nil {
		return
	}
	pdata := PointerBucket.Get(post.DBKey())
	if pdata == nil {
		return
	}
	p := Pointer{}
	json.Unmarshal(pdata, &p)
	FeedsBucket := tx.Bucket([]byte("feeds"))
	if FeedsBucket == nil {
		return
	}
	FeedBucket := FeedsBucket.Bucket(p.Author.DBKey())
	if FeedBucket == nil {
		return
	}
	LogBucket := FeedBucket.Bucket([]byte("log"))
	if LogBucket == nil {
		return
	}
	msgdata := LogBucket.Get(itob(p.Sequence))
	if msgdata == nil {
		return
	}
	json.Unmarshal(msgdata, &m)
	return
}

var AddMessageHooks = map[string]func(m *SignedMessage, tx *bolt.Tx) error{}

func (f *Feed) AddMessage(m *SignedMessage) error {
	f.addChan <- m
	return nil
}

func (f *Feed) addMessage(m *SignedMessage) error {
	if m.Author != f.ID {
		return fmt.Errorf("Wrong feed")
	}
	if f.store.Get(nil, m.Key()) != nil {
		return nil
	}
	err := m.Verify(f)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = f.store.db.Update(func(tx *bolt.Tx) error {
		FeedsBucket, err := tx.CreateBucketIfNotExists([]byte("feeds"))
		if err != nil {
			return err
		}
		FeedBucket, err := FeedsBucket.CreateBucketIfNotExists(f.ID.DBKey())
		if err != nil {
			return err
		}
		FeedLogBucket, err := FeedBucket.CreateBucketIfNotExists([]byte("log"))
		if err != nil {
			return err
		}
		FeedLogBucket.FillPercent = 1
		buf, err := Encode(m)
		if err != nil {
			return err
		}
		err = FeedLogBucket.Put(itob(m.Sequence), buf)
		if err != nil {
			return err
		}
		LogBucket, err := tx.CreateBucketIfNotExists([]byte("log"))
		if err != nil {
			return err
		}
		LogBucket.FillPercent = 1
		seq, err := LogBucket.NextSequence()
		if err != nil {
			return err
		}
		err = LogBucket.Put(itob(int(seq)), m.Key().DBKey())
		if err != nil {
			return err
		}
		PointerBucket, err := tx.CreateBucketIfNotExists([]byte("pointer"))
		if err != nil {
			return err
		}
		pointer := Pointer{Author: m.Author, Sequence: m.Sequence}
		buf, _ = json.Marshal(pointer)
		err = PointerBucket.Put(m.Key().DBKey(), buf)
		if err != nil {
			return err
		}
		for module, hook := range AddMessageHooks {
			err = hook(m, tx)
			if err != nil {
				return fmt.Errorf("Bolt %s hook: %s", module, err)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Bolt: ", err)
		return err
	}
	f.Topic.Send <- m
	return nil
}

var RebuildClearHooks = map[string]func(tx *bolt.Tx) error{}

func (ds *DataStore) RebuildAll() {
	log.Println("Starting rebuild of all indexes")
	count := 0
	ds.db.Update(func(tx *bolt.Tx) error {
		for module, hook := range RebuildClearHooks {
			err := hook(tx)
			if err != nil {
				return fmt.Errorf("Bolt %s hook: %s", module, err)
			}
		}

		LogBucket, err := tx.CreateBucketIfNotExists([]byte("log"))
		if err != nil {
			return err
		}
		cursor := LogBucket.Cursor()
		_, v := cursor.First()
		for v != nil {
			for module, hook := range AddMessageHooks {
				err = hook(ds.Get(tx, DBRef(v)), tx)
				if err != nil {
					return fmt.Errorf("Bolt %s hook: %s", module, err)
				}
			}
			count++
			_, v = cursor.Next()
		}
		return nil
	})
	log.Println("Finished rebuild of all modules")
	log.Println("Reindexed", count, "posts")
}

func (ds *DataStore) Rebuild(module string) {
	log.Println("Starting rebuild of", module)
	count := 0
	ds.db.Update(func(tx *bolt.Tx) error {
		if clear, ok := RebuildClearHooks[module]; ok {
			err := clear(tx)
			if err != nil {
				return err
			}
		}

		LogBucket, err := tx.CreateBucketIfNotExists([]byte("log"))
		if err != nil {
			return err
		}
		cursor := LogBucket.Cursor()
		_, v := cursor.First()
		for v != nil {
			AddMessageHooks[module](ds.Get(tx, DBRef(v)), tx)
			count++
			_, v = cursor.Next()
		}
		return nil
	})
	log.Println("Finished rebuild of", module)
	log.Println("Reindexed", count, "messages")
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
		FeedBucket := FeedsBucket.Bucket(f.ID.DBKey())
		if FeedBucket == nil {
			return nil
		}
		FeedLogBucket := FeedBucket.Bucket([]byte("log"))
		if FeedLogBucket == nil {
			return nil
		}
		cur := FeedLogBucket.Cursor()
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
			FeedBucket := FeedsBucket.Bucket(f.ID.DBKey())
			if FeedBucket == nil {
				return nil
			}
			FeedLogBucket := FeedBucket.Bucket([]byte("log"))
			if FeedLogBucket == nil {
				return nil
			}
			err := FeedLogBucket.ForEach(func(k, v []byte) error {
				var m *SignedMessage
				json.Unmarshal(v, &m)
				if m.Sequence < seq {
					return nil
				}
				seq = m.Sequence
				select {
				case c <- m:
				case <-time.After(1000 * time.Millisecond):
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
