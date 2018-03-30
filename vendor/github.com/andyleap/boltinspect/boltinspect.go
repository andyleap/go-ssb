// BoltInspect project boltinspect.go
package boltinspect

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/boltdb/bolt"
)

type BoltInspect struct {
	db        *bolt.DB
	Templates *template.Template
}

func New(db *bolt.DB) *BoltInspect {
	tpls := template.New("inspect.tpl")
	template.Must(tpls.Parse(inspectTemplate))
	template.Must(tpls.New("bucket.tpl").Parse(bucketTemplate))
	template.Must(tpls.New("item.tpl").Parse(itemTemplate))
	template.Must(tpls.New("view.tpl").Parse(viewTemplate))

	return &BoltInspect{
		db:        db,
		Templates: tpls,
	}
}

type bucketData struct {
	ID string
}

type itemData struct {
	ID      string
	Content string
}

func (bi *BoltInspect) InspectEndpoint(rw http.ResponseWriter, req *http.Request) {
	bucket := req.FormValue("bucket")
	buckets := strings.Split(bucket, "/")
	bucketsbyte := make([][]byte, 0)
	for _, v := range buckets {
		if v != "" {
			bucketsbyte = append(bucketsbyte, []byte(v))
		}
	}
	bucketdata := make([]*bucketData, 0)
	itemdata := make([]*itemData, 0)
	fmt.Println(bucketsbyte)
	bi.db.View(func(tx *bolt.Tx) error {
		if len(bucketsbyte) == 0 {
			tx.ForEach(func(k []byte, v *bolt.Bucket) error {
				bucketdata = append(bucketdata, &bucketData{
					ID: string(k),
				})
				return nil
			})
			bi.Templates.ExecuteTemplate(rw, "inspect.tpl", struct {
				Buckets []*bucketData
				Items   []*itemData
			}{
				bucketdata,
				itemdata,
			})
			return nil
		}
		bucket := tx.Bucket(bucketsbyte[0])
		idPrefix := string(bucketsbyte[0])
		for _, b := range bucketsbyte[1:] {
			if bucket.Get(b) == nil {
				bucket = bucket.Bucket(b)
				idPrefix = idPrefix + "/" + string(b)
			} else {
				bi.Templates.ExecuteTemplate(rw, "view.tpl", &itemData{
					ID:      idPrefix + "/" + string(b),
					Content: string(bucket.Get(b)),
				})
				return nil
			}
		}
		bucket.ForEach(func(k, v []byte) error {
			if v == nil {
				bucketdata = append(bucketdata, &bucketData{
					ID: idPrefix + "/" + string(k),
				})
			} else {
				itemdata = append(itemdata, &itemData{
					ID:      idPrefix + "/" + string(k),
					Content: string(v),
				})
			}
			return nil
		})
		bi.Templates.ExecuteTemplate(rw, "inspect.tpl", struct {
			Buckets []*bucketData
			Items   []*itemData
		}{
			bucketdata,
			itemdata,
		})
		return nil

	})

}

var inspectTemplate = `
<html><head></head><body>
<ul>
	<li><h1>Buckets</h1></li>
	{{range .Buckets}}
	<li>{{template "bucket.tpl" .}}</li>
	{{end}}
	<li><h1>Items</h1></li>
	{{range .Items}}
	<li>{{template "item.tpl" .}}</li>
	{{end}}
</ul>
</body></html>
`

var bucketTemplate = `
<a href="?bucket={{.ID}}">{{.ID}}</a>
`

var itemTemplate = `
<a href="?bucket={{.ID}}">{{.ID}}</a>
`

var viewTemplate = `
<html><head></head><body>
<h1>{{.ID}}</h1>
{{.Content}}
</body></html>
`
