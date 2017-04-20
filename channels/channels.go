package channels

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/social"

	"github.com/boltdb/bolt"
)

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func init() {
	ssb.AddMessageHooks["channels"] = func(m *ssb.SignedMessage, tx *bolt.Tx) error {
		_, mb := m.DecodeMessage()
		if mbr, ok := mb.(*social.Post); ok {
			if mbr.Channel != "" {
				channelsBucket, err := tx.CreateBucketIfNotExists([]byte("channels"))
				if err != nil {
					return err
				}
				channelBucket, err := channelsBucket.CreateBucketIfNotExists([]byte(mbr.Channel))
				if err != nil {
					return err
				}
				logBucket, err := channelBucket.CreateBucketIfNotExists([]byte("log"))
				if err != nil {
					return err
				}
				logBucket.FillPercent = 1
				seq, err := logBucket.NextSequence()
				if err != nil {
					return err
				}
				logBucket.Put(itob(int(seq)), []byte(m.Key()))

				timeBucket, err := channelBucket.CreateBucketIfNotExists([]byte("time"))
				timeBucket.FillPercent = 1
				if err != nil {
					return err
				}
				i := int(m.Timestamp * float64(time.Millisecond))
				for timeBucket.Get(itob(i)) != nil {
					i++
				}
				timeBucket.Put(itob(i), []byte(m.Key()))
			}
		}
		return nil
	}
}

func GetChannelLatest(ds *ssb.DataStore, channel string, num int) (msgs []*ssb.SignedMessage) {
	ds.DB().View(func(tx *bolt.Tx) error {
		channelsBucket := tx.Bucket([]byte("channels"))
		if channelsBucket == nil {
			return nil
		}
		channelBucket := channelsBucket.Bucket([]byte(channel))
		if channelBucket == nil {
			return nil
		}
		timeBucket := channelBucket.Bucket([]byte("time"))
		if timeBucket == nil {
			return nil
		}
		cursor := timeBucket.Cursor()
		_, v := cursor.Last()
		for l1 := 0; l1 < num; l1++ {
			fmt.Println(l1)
			if v == nil {
				break
			}
			m := ds.Get(tx, ssb.Ref(v))
			msgs = append(msgs, m)
			_, v = cursor.Prev()
		}
		return nil
	})
	return
}
