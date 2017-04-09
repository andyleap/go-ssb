package ssb

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
)

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

type FeedStore struct {
	db *bolt.DB

	feedlock sync.Mutex
	feeds    map[Ref]*Feed
}

type Feed struct {
	store *FeedStore
	ID    Ref
}

func OpenFeedStore(path string) (*FeedStore, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &FeedStore{db: db, feeds: map[Ref]*Feed{}}, nil
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
	feed := &Feed{store: fs, ID: feedID}
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
