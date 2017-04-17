package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"github.com/andyleap/go-ssb"
)

var ContentTemplates = template.Must(template.ParseGlob("templates/content/*.tpl"))

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
	mchan := datastore.GetFeed(datastore.PrimaryRef).Log(0, false)
	messages := []*ssb.SignedMessage{}
	for m := range mchan {
		log.Println(m)
		messages = append(messages, m)
	}
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
