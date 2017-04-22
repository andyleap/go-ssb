package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/russross/blackfriday"

	"github.com/andyleap/boltinspect"
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

var ChannelTemplate = template.Must(template.New("channel").Funcs(template.FuncMap{
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
<form action="/publish/post" method="post">
<textarea name="text"></textarea><br>
<input type="hidden" name="channel" value="{{.Channel}}">
<input type="hidden" name="returnto" value="/channel?channel={{.Channel}}">
<input type="submit" value="Publish!">
</form>
<hr>
{{range .Messages}}
{{RenderContent .}}<hr>
{{end}}
</body>
</html>`))

var AdminTemplate = template.Must(template.New("admin").Parse(`
<html>
<head>
</head>
<body>
<a href="/rebuild?module=all">all</a><br>
{{range .Modules}}
<a href="/rebuild?module={{.}}">{{.}}</a><br>
{{end}}
</body>
</html>`))

var PostTemplate = template.Must(template.New("post").Funcs(template.FuncMap{
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
<form action="/publish/post" method="post">
<textarea name="text"></textarea><br>
<input type="hidden" name="channel" value="{{.Content.Channel}}">
<input type="hidden" name="branch" value="{{.Message.Key}}">
<input type="hidden" name="root" value="{{if eq .Content.Root ""}}{{.Message.Key}}{{else}}{{.Content.Root}}{{end}}">
<input type="hidden" name="returnto" value="/post?id={{.Message.Key | urlquery}}">
<input type="submit" value="Publish!">
</form>
<hr>
{{RenderContent .Message}}<hr>
</body>
</html>`))

func init() {
	log.Println(ContentTemplates.DefinedTemplates())
}

func RegisterWebui() {
	bi := boltinspect.New(datastore.DB())

	http.HandleFunc("/bolt", bi.InspectEndpoint)

	//http.HandleFunc("/", Index)
	http.HandleFunc("/channel", Channel)
	http.HandleFunc("/post", Post)
	http.HandleFunc("/publish/post", PublishPost)

	http.HandleFunc("/admin", Admin)
	http.HandleFunc("/rebuild", Rebuild)

	go http.ListenAndServe(":9823", nil)
}

func PublishPost(rw http.ResponseWriter, req *http.Request) {
	p := &social.Post{}
	p.Type = "post"
	p.Root = ssb.ParseRef(req.FormValue("root"))
	p.Branch = ssb.ParseRef(req.FormValue("branch"))
	p.Channel = req.FormValue("channel")
	p.Text = req.FormValue("text")
	datastore.GetFeed(datastore.PrimaryRef).PublishMessage(p)
	http.Redirect(rw, req, req.FormValue("returnto"), http.StatusSeeOther)
}

func Rebuild(rw http.ResponseWriter, req *http.Request) {
	module := req.FormValue("module")
	if module != "" {
		if module == "all" {
			datastore.RebuildAll()
		} else {
			datastore.Rebuild(module)
		}
	}
	http.Redirect(rw, req, "/admin", http.StatusSeeOther)
}

func Admin(rw http.ResponseWriter, req *http.Request) {
	modules := []string{}
	for module := range ssb.AddMessageHooks {
		modules = append(modules, module)
	}
	err := AdminTemplate.Execute(rw, struct {
		Modules []string
	}{
		modules,
	})
	if err != nil {
		log.Println(err)
	}
}

func Index(rw http.ResponseWriter, req *http.Request) {
	messages := channels.GetChannelLatest(datastore, "golang", 20)
	log.Println(messages)
	err := ChannelTemplate.Execute(rw, struct {
		Messages []*ssb.SignedMessage
		Channel  string
	}{
		messages,
		"golang",
	})
	if err != nil {
		log.Println(err)
	}
}

func Post(rw http.ResponseWriter, req *http.Request) {
	post := req.FormValue("id")
	if post == "" {
		Index(rw, req)
		return
	}
	message := datastore.Get(nil, ssb.ParseRef(post))
	if message == nil {
		http.NotFound(rw, req)
		return
	}
	_, content := message.DecodeMessage()
	p, ok := content.(*social.Post)
	if !ok {
		Index(rw, req)
		return
	}
	log.Println(message)
	err := PostTemplate.Execute(rw, struct {
		Message *ssb.SignedMessage
		Content *social.Post
	}{
		message,
		p,
	})
	if err != nil {
		log.Println(err)
	}
}

func Channel(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Channel Request!")
	channel := req.FormValue("channel")
	if channel == "" {
		Index(rw, req)
		return
	}
	messages := channels.GetChannelLatest(datastore, channel, 100)
	log.Println(messages)
	err := ChannelTemplate.Execute(rw, struct {
		Messages []*ssb.SignedMessage
		Channel  string
	}{
		messages,
		channel,
	})
	if err != nil {
		log.Println(err)
	}
}
