package main

import (
	"net/url"
	"io"
	"bytes"
	"html/template"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/boltdb/bolt"
	"github.com/russross/blackfriday"

	"github.com/andyleap/boltinspect"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/channels"
	"github.com/andyleap/go-ssb/search"
	"github.com/andyleap/go-ssb/social"
	"github.com/andyleap/go-ssb/blobs"
)

var ContentTemplates = template.New("content")

type SSBRenderer struct {
	blackfriday.Renderer
}

func (ssbr *SSBRenderer) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	r := ssb.ParseRef(string(link))
	switch r.Type {
		case ssb.RefBlob:
		link = []byte("/blob?id=" + url.QueryEscape(r.String()))
	}
	ssbr.Renderer.Image(out, link, title, alt)
}

func RenderMarkdown(input []byte) []byte {
commonHtmlFlags := 0 |
		blackfriday.HTML_USE_XHTML |
		blackfriday.HTML_USE_SMARTYPANTS |
		blackfriday.HTML_SMARTYPANTS_FRACTIONS |
		blackfriday.HTML_SMARTYPANTS_DASHES |
		blackfriday.HTML_SMARTYPANTS_LATEX_DASHES

	commonExtensions := 0 |
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_TABLES |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH |
		blackfriday.EXTENSION_SPACE_HEADERS |
		blackfriday.EXTENSION_HEADER_IDS |
		blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
		blackfriday.EXTENSION_DEFINITION_LISTS
	// set up the HTML renderer
	renderer := &SSBRenderer{blackfriday.HtmlRenderer(commonHtmlFlags, "", "")}
	options := blackfriday.Options{
		Extensions: commonExtensions}

	return blackfriday.MarkdownOptions(input, renderer, options)
}

func init() {
	template.Must(ContentTemplates.Funcs(template.FuncMap{
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
			return template.HTML(RenderMarkdown([]byte(markdown)))
		},
		"GetMessage": func(ref ssb.Ref) *ssb.SignedMessage {
			return datastore.Get(nil, ref)
		},
		"RenderContent": func(m *ssb.SignedMessage) template.HTML {
			if m == nil {
				return ""
			}
			t, md := m.DecodeMessage()
			buf := &bytes.Buffer{}
			err := ContentTemplates.ExecuteTemplate(buf, t+".tpl", struct {
				Message  *ssb.SignedMessage
				Content  interface{}
				Embedded bool
			}{m, md, true})
			if err != nil {
				log.Println(err)
			}
			return template.HTML(buf.String())
		},
	}).ParseGlob("templates/content/*.tpl"))
}

var ChannelTemplate = template.Must(template.New("channel").Funcs(template.FuncMap{
	"RenderContent": func(m *ssb.SignedMessage) template.HTML {
		t, md := m.DecodeMessage()
		buf := &bytes.Buffer{}
		err := ContentTemplates.ExecuteTemplate(buf, t+".tpl", struct {
			Message  *ssb.SignedMessage
			Content  interface{}
			Embedded bool
		}{m, md, false})
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

var SearchTemplate = template.Must(template.New("search").Funcs(template.FuncMap{
	"RenderContent": func(m *ssb.SignedMessage) template.HTML {
		t, md := m.DecodeMessage()
		buf := &bytes.Buffer{}
		err := ContentTemplates.ExecuteTemplate(buf, t+".tpl", struct {
			Message  *ssb.SignedMessage
			Content  interface{}
			Embedded bool
		}{m, md, false})
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

var AdminTemplate = template.Must(template.New("admin").Parse(`
<html>
<head>
</head>
<body>
<table>
<tr><th>key</th><th>size</th></tr>
{{range $b, $size := .DBSize}}
<tr><td>{{$b}}</td><td style="text-align: right;">{{$size}}</td>
{{end}}
</table><br>
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
			Message  *ssb.SignedMessage
			Content  interface{}
			Embedded bool
		}{m, md, false})
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
{{RenderContent .Message}}
<hr>
<form action="/publish/post" method="post">
<textarea name="text"></textarea><br>
<input type="hidden" name="channel" value="{{.Content.Channel}}">
<input type="hidden" name="branch" value="{{.Message.Key}}">
<input type="hidden" name="root" value="{{if eq .Content.Root.Type 0}}{{.Message.Key}}{{else}}{{.Content.Root}}{{end}}">
<input type="hidden" name="returnto" value="/post?id={{.Message.Key | urlquery}}">
<input type="submit" value="Publish!">
</form>
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
	http.HandleFunc("/search", Search)
	http.HandleFunc("/publish/post", PublishPost)

	http.HandleFunc("/admin", Admin)
	http.HandleFunc("/rebuild", Rebuild)
	
	http.HandleFunc("/blob", Blob)

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

func calcSize(tx *bolt.Tx, b *bolt.Bucket) (size int) {
	b.ForEach(func(k, v []byte) error {
		size += len(k)
		if v == nil {
			size += calcSize(tx, b.Bucket(k))
		} else {
			size += len(v)
		}
		return nil
	})
	return
}

func Admin(rw http.ResponseWriter, req *http.Request) {
	size := map[string]int{}
	datastore.DB().View(func(tx *bolt.Tx) error {
		tx.ForEach(func(k []byte, b *bolt.Bucket) error {
			size[string(k)] = calcSize(tx, b)
			return nil
		})
		return nil
	})

	modules := []string{}
	for module := range ssb.AddMessageHooks {
		modules = append(modules, module)
	}
	err := AdminTemplate.Execute(rw, struct {
		Modules []string
		DBSize  map[string]int
	}{
		modules,
		size,
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
	channel := req.FormValue("channel")
	if channel == "" {
		Index(rw, req)
		return
	}
	messages := channels.GetChannelLatest(datastore, channel, 100)
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

func Search(rw http.ResponseWriter, req *http.Request) {
	query := req.FormValue("q")
	if query == "" {
		Index(rw, req)
		return
	}
	messages := search.Search(datastore, query, 50)
	err := SearchTemplate.Execute(rw, struct {
		Messages []*ssb.SignedMessage
	}{
		messages,
	})
	if err != nil {
		log.Println(err)
	}
}

func Blob(rw http.ResponseWriter, req *http.Request) {
	id := req.FormValue("id")
	if id == "" {
		http.NotFound(rw, req)
		return
	}
	r := ssb.ParseRef(id)
	bs := datastore.ExtraData("blobStore").(*blobs.BlobStore)
	if !bs.Has(r) {
		bs.Want(r)
		bs.WaitFor(r)
	}
	rc := bs.Get(r)
	defer rc.Close()
	io.Copy(rw, rc)
}
