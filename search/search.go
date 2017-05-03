package search

import (
	"strings"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/social"
	"github.com/boltdb/bolt"
)

func Search(ds *ssb.DataStore, term string, max int) (found []*ssb.SignedMessage) {
	ds.DB().View(func(tx *bolt.Tx) error {

		LogBucket := tx.Bucket([]byte("log"))
		if LogBucket == nil {
			return nil
		}
		cursor := LogBucket.Cursor()
		_, v := cursor.Last()
		for v != nil {
			m := ds.Get(tx, ssb.DBRef(v))
			_, md := m.DecodeMessage()
			if post, ok := md.(*social.Post); ok {
				if strings.Contains(post.Text, term) {
					found = append(found, m)
					if max > 0 && len(found) >= max {
						return nil
					}
				}
			}
			_, v = cursor.Prev()
		}
		return nil
	})
	return
}
