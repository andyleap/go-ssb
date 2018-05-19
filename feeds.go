package ssb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"cryptoscope.co/go/secretstream/secrethandshake"
	"github.com/boltdb/bolt"
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

var initMethods = []func(ds *DataStore){}

func RegisterInit(f func(ds *DataStore)) {
	initMethods = append(initMethods, f)
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

func init() {
	RegisterInit(func(ds *DataStore) {
		ds.RegisterMethod("feed.Publish", func(feed Ref, message interface{}) error {
			return ds.GetFeed(feed).PublishMessage(message)
		})
		ds.RegisterMethod("feed.Latest", func(feed Ref) *SignedMessage {
			return ds.GetFeed(feed).Latest()
		})
	})
	/*AddMessageHooks["recompress"] = func(m *SignedMessage, tx *bolt.Tx) error {
		FeedsBucket, err := tx.CreateBucketIfNotExists([]byte("feeds"))
		if err != nil {
			return err
		}
		FeedBucket, err := FeedsBucket.CreateBucketIfNotExists(m.Author.DBKey())
		if err != nil {
			return err
		}
		FeedLogBucket, err := FeedBucket.CreateBucketIfNotExists([]byte("log"))
		if err != nil {
			return err
		}
		FeedLogBucket.FillPercent = 1
		buf := m.Compress()
		err = FeedLogBucket.Put(itob(m.Sequence), buf)
		if err != nil {
			return err
		}
		return nil
	}*/
}

func (ds *DataStore) RegisterMethod(name string, method interface{}) {
	RPCMethods, ok := ds.ExtraData("RPCMethods").(map[string]interface{})
	if !ok {
		RPCMethods = map[string]interface{}{}
	}
	RPCMethods[name] = method
	ds.SetExtraData("RPCMethods", RPCMethods)
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

func (ds *DataStore) Close() {
	err := ds.db.Close()
	if err != nil {
		log.Println("error closing db:", err)
	}
}

type Feed struct {
	store *DataStore
	ID    Ref

	Topic     *MessageTopic
	LatestSeq int
	SeqLock   sync.Mutex

	addChan chan *SignedMessage

	waiting       map[int]*SignedMessage
	waitingLock   sync.Mutex
	waitingSignal *sync.Cond
}

type Pointer struct {
	Sequence int
	LogKey   int
	Author   []byte
}

func (p Pointer) Marshal() []byte {
	buf := make([]byte, len(p.Author)+16)
	binary.BigEndian.PutUint64(buf[0:], uint64(p.Sequence))
	binary.BigEndian.PutUint64(buf[8:], uint64(p.LogKey))
	copy(buf[16:], p.Author)
	return buf
}

func (p *Pointer) Unmarshal(buf []byte) {
	p.Author = make([]byte, len(buf)-16)
	p.Sequence = int(binary.BigEndian.Uint64(buf[0:]))
	p.LogKey = int(binary.BigEndian.Uint64(buf[8:]))
	copy(p.Author, buf[16:])
}

func OpenDataStore(path string, primaryKey *secrethandshake.EdKeyPair) (*DataStore, error) {
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
	ds.PrimaryKey = primaryKey
	ds.PrimaryRef, _ = NewRef(RefFeed, ds.PrimaryKey.Public[:], RefAlgoEd25519)
	ds.Keys[ds.PrimaryRef] = &SignerEd25519{ed25519.PrivateKey(ds.PrimaryKey.Secret[:])}

	for _, im := range initMethods {
		im(ds)
	}

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
	feed := &Feed{
		store:         ds,
		ID:            feedID,
		Topic:         NewMessageTopic(),
		addChan:       make(chan *SignedMessage, 10),
		waiting:       map[int]*SignedMessage{},
		waitingSignal: sync.NewCond(&sync.Mutex{}),
	}
	go func() {
		for m := range feed.addChan {
			feed.waitingLock.Lock()
			feed.SeqLock.Lock()
			if m.Sequence > feed.LatestSeq {
				feed.waiting[m.Sequence] = m
			}
			feed.SeqLock.Unlock()
			feed.waitingLock.Unlock()
			feed.waitingSignal.Broadcast()
		}
	}()
	go feed.processMessageQueue()
	m := feed.Latest()
	if m != nil {
		feed.LatestSeq = m.Sequence
	}

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
	p.Unmarshal(pdata)
	FeedsBucket := tx.Bucket([]byte("feeds"))
	if FeedsBucket == nil {
		return
	}
	FeedBucket := FeedsBucket.Bucket(p.Author)
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
	m = DecompressMessage(msgdata)
	return
}

func GetMsg(tx *bolt.Tx, post Ref) (m *SignedMessage) {
	PointerBucket := tx.Bucket([]byte("pointer"))
	if PointerBucket == nil {
		return
	}
	pdata := PointerBucket.Get(post.DBKey())
	if pdata == nil {
		return
	}
	p := Pointer{}
	p.Unmarshal(pdata)
	FeedsBucket := tx.Bucket([]byte("feeds"))
	if FeedsBucket == nil {
		return
	}
	FeedBucket := FeedsBucket.Bucket(p.Author)
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
	m = DecompressMessage(msgdata)
	return
}

var AddMessageHooks = map[string]func(m *SignedMessage, tx *bolt.Tx) error{}

func (f *Feed) AddMessage(m *SignedMessage) error {
	if m != nil {
		f.addChan <- m
	}
	return nil
}

func (f *Feed) processMessageQueue() {
	for {
		f.waitingSignal.L.Lock()
		f.waitingSignal.Wait()
		f.waitingSignal.L.Unlock()
		newMsgs := []*SignedMessage{}
		err := f.store.db.Update(func(tx *bolt.Tx) error {
			f.waitingLock.Lock()
			f.SeqLock.Lock()
			defer func() {
				f.SeqLock.Unlock()
				f.waitingLock.Unlock()
			}()
			for {
				m, ok := f.waiting[f.LatestSeq+1]
				delete(f.waiting, f.LatestSeq+1)
				if !ok {
					break
				}

				if m.Author != f.ID {
					continue
				}
				if f.store.Get(nil, m.Key()) != nil {
					continue
				}
				err := m.Verify(tx, f)
				if err != nil {
					//fmt.Println(err)
					//fmt.Println((string(m.Message.Content)))
					fmt.Print("-")
					return err
				}
				err = f.addMessage(tx, m)
				if err != nil {
					fmt.Println("Bolt: ", err)
					return err
				}

				f.LatestSeq = m.Sequence
				fmt.Print("*")
				newMsgs = append(newMsgs, m)
			}
			return nil
		})
		if err != nil {
			continue
		}
		for _, m := range newMsgs {
			f.Topic.Send <- m
		}
	}
}

func (f *Feed) addMessage(tx *bolt.Tx, m *SignedMessage) error {
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
	buf := m.Compress()
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
	pointer := Pointer{Sequence: m.Sequence, LogKey: int(seq), Author: m.Author.DBKey()}
	buf = pointer.Marshal()
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

func (ds *DataStore) LatestCountFiltered(num int, start int, filter map[Ref]int) (msgs []*SignedMessage) {
	ds.db.View(func(tx *bolt.Tx) error {
		LogBucket := tx.Bucket([]byte("log"))
		if LogBucket == nil {
			return nil
		}
		cur := LogBucket.Cursor()
		_, val := cur.Last()
		for len(msgs) < num {
			for i := 0; i < start; i++ {
				_, val = cur.Prev()
				if val == nil {
					break
				}
			}
			if val == nil {
				break
			}
			msg := ds.Get(tx, DBRef(val))

			if _, ok := filter[msg.Author]; ok && msg.Type() != "" {
				msgs = append(msgs, msg)
			}
			_, val = cur.Prev()
		}
		return nil
	})
	return
}

func (f *Feed) PublishMessage(body interface{}) error {
	content, _ := Encode(body)

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
	c := f.Topic.Register(nil, true)
	err := f.AddMessage(sm)
	if err != nil {
		return err
	}

	for newm := range c {
		if newm.Key() == sm.Key() {
			f.Topic.Unregister(c)
			return nil
		}
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
		m = DecompressMessage(val)
		return nil
	})
	return
}

func (f *Feed) LatestCount(num int, start int) (msgs []*SignedMessage) {
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
		for i := 0; i < start; i++ {
			_, val = cur.Prev()
		}
		for l1 := 0; l1 < num; l1++ {
			if val == nil {
				break
			}
			msg := DecompressMessage(val)
			if msg.Type() != "" {
				msgs = append(msgs, msg)
			}
			_, val = cur.Prev()
		}
		return nil
	})
	return
}

func (f *Feed) GetSeq(tx *bolt.Tx, seq int) (m *SignedMessage) {
	if tx == nil {
		tx, _ = f.store.db.Begin(false)
		defer tx.Rollback()
	}
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
	val := FeedLogBucket.Get(itob(seq))
	if val == nil {
		return nil
	}
	m = DecompressMessage(val)
	return m
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
				m := DecompressMessage(v)
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
