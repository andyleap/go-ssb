package ssb

import (
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

type Relation struct {
	Following bool
	Blocking  bool
}

func (fs *FeedStore) HandleGraph() {
	c := fs.Topic.Register(nil, true)

	for m := range c {
		mb := m.DecodeMessage()
		if mbc, ok := mb.(*Contact); ok {
			fmt.Println(mbc)
			fs.db.Update(func(tx *bolt.Tx) error {
				GraphBucket, err := tx.CreateBucketIfNotExists([]byte("graph"))
				if err != nil {
					return err
				}
				FeedBucket, err := GraphBucket.CreateBucketIfNotExists([]byte(m.Author))
				var r Relation
				json.Unmarshal(FeedBucket.Get([]byte(mbc.Contact)), &r)
				if err != nil {
					return err
				}
				if mbc.Following != nil {
					r.Following = *mbc.Following
				}
				if mbc.Blocking != nil {
					r.Blocking = *mbc.Blocking
				}
				buf, _ := json.Marshal(r)
				err = FeedBucket.Put([]byte(mbc.Contact), buf)
				if err != nil {
					return err
				}
				return nil
			})
		}
	}
}

func (fs *FeedStore) GetFollows(feed Ref, depth int) (follows map[Ref]int) {
	follows = map[Ref]int{}
	follows[feed] = 0
	fs.db.View(func(tx *bolt.Tx) error {
		GraphBucket := tx.Bucket([]byte("graph"))
		if GraphBucket == nil {
			return nil
		}
		for l1 := 0; l1 < depth; l1++ {
			for k, v := range follows {
				if v == l1 {
					FeedBucket := GraphBucket.Bucket([]byte(k))
					if FeedBucket == nil {
						continue
					}
					FeedBucket.ForEach(func(k, v []byte) error {
						if _, ok := follows[Ref(k)]; !ok {
							var r Relation
							json.Unmarshal(v, &r)
							if r.Following {
								follows[Ref(k)] = l1 + 1
							}
						}
						return nil
					})
				}
			}
		}
		return nil
	})
	return
}
