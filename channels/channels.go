package channels

import (
	"encoding/binary"
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

type Channel struct {
	ssb.MessageBody
	Channel    string `json:"channel"`
	Subscribed bool   `json:"subscribed"`
}

func init() {
	ssb.MessageTypes["channel"] = func(mb ssb.MessageBody) interface{} { return &Channel{MessageBody: mb} }
	ssb.RebuildClearHooks["channels"] = func(tx *bolt.Tx) error {
		tx.DeleteBucket([]byte("channels"))
		return nil
	}
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
				logBucket.Put(itob(int(seq)), m.Key().DBKey())

				timeBucket, err := channelBucket.CreateBucketIfNotExists([]byte("time"))
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

func GetChannelLatest(ds *ssb.DataStore, channel string, num int, start int) (msgs []*ssb.SignedMessage) {
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
        for i := 0; i < start; i++ {
            _, v = cursor.Prev()
            if v == nil {
                break
            }
        }
		for l1 := 0; l1 < num; l1++ {
			if v == nil {
				break
			}
			m := ds.Get(tx, ssb.DBRef(v))
			msgs = append(msgs, m)
			_, v = cursor.Prev()
		}
		return nil
	})
	return
}
