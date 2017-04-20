package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/russross/blackfriday"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/channels"
	"github.com/andyleap/go-ssb/social"
)

var ContentTemplates = template.Must(template.New("content").Funcs(template.FuncMap{
	"GetAbout": func(ref ssb.Ref) (a *social.About) {
		datastore.DB().View(func(tx *bolt.Tx) error {
			a = social.GetAbout(tx, ref)
			return nil
		})
		return
	},
	"RenderJSTime": func(timestamp float64) string {
		t := time.Unix(0, int64(timestamp*float64(time.Millisecond))).Local()
		return t.Format(time.ANSIC)
	},
	"Markdown": func(markdown string) template.HTML {
		return template.HTML(blackfriday.MarkdownCommon([]byte(markdown)))
	},
}).ParseGlob("templates/content/*.tpl"))

var IndexTemplate = template.Must(template.New("index").Funcs(template.FuncMap{
	"RenderContent": func(m *ssb.SignedMessage) template.HTML {
		t, md := m.DecodeMessage()
		buf := &bytes.Buffer{}
		err := ContentTemplates.ExecuteTemplate(buf, t+".tpl", struct {
			Message *ssb.SignedMessage
			Content interface{}
		}{m, md})
		if err != nil {
			log.Println(err)
		}
		return template.HTML(buf.String())
	},
}).Parse(`
<html>
<head>
</head>
<body>
{{range .Messages}}
{{RenderContent .}}<hr>
{{end}}
</body>
</html>`))

func init() {
	log.Println(ContentTemplates.DefinedTemplates())
}

func Index(rw http.ResponseWriter, req *http.Request) {
	messages := channels.GetChannelLatest(datastore, "golang", 20)
	log.Println(messages)
	err := IndexTemplate.Execute(rw, struct {
		Messages []*ssb.SignedMessage
	}{
		messages,
	})
	if err != nil {
		log.Println(err)
	}
}

func Channel(rw http.ResponseWriter, req *http.Request) {
	channel := req.FormValue("channel")
	if channel == "" {
		Index(rw, req)
		return
	}
	messages := channels.GetChannelLatest(datastore, channel, 20)
	log.Println(messages)
	err := IndexTemplate.Execute(rw, struct {
		Messages []*ssb.SignedMessage
	}{
		messages,
	})
	if err != nil {
		log.Println(err)
	}
}
