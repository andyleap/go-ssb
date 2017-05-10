package main

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"time"

	"github.com/boltdb/bolt"
	"github.com/russross/blackfriday"

	"github.com/andyleap/boltinspect"

	"github.com/andyleap/go-ssb"
	"github.com/andyleap/go-ssb/blobs"
	"github.com/andyleap/go-ssb/channels"
	"github.com/andyleap/go-ssb/graph"
	"github.com/andyleap/go-ssb/search"
	"github.com/andyleap/go-ssb/social"
)

var ContentTemplates = template.New("content")

type SSBRenderer struct {
	blackfriday.Renderer
}

func (ssbr *SSBRenderer) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	r := ssb.ParseRef(string(link))
	switch r.Type {
	case ssb.RefBlob:
		link = []byte("/blob?id=" + url.QueryEscape(r.String()))
	case ssb.RefMessage:
		link = []byte("/post?id=" + url.QueryEscape(r.String()))
	case ssb.RefFeed:
		link = []byte("/feed?id=" + url.QueryEscape(r.String()))
	}
	if link[0] == '#' {
		link = []byte("/channel?channel=" + url.QueryEscape(string(link[1:])))
	}
	ssbr.Renderer.AutoLink(out, link, kind)
}
func (ssbr *SSBRenderer) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	r := ssb.ParseRef(string(link))
	switch r.Type {
	case ssb.RefBlob:
		link = []byte("/blob?id=" + url.QueryEscape(r.String()))
	case ssb.RefMessage:
		link = []byte("/post?id=" + url.QueryEscape(r.String()))
	case ssb.RefFeed:
		link = []byte("/feed?id=" + url.QueryEscape(r.String()))
	}
	if link[0] == '#' {
		link = []byte("/channel?channel=" + url.QueryEscape(string(link[1:])))
	}
	ssbr.Renderer.Link(out, link, title, content)
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
		"Avatar": func(ref ssb.Ref) template.HTML {
			if ref.Type != ssb.RefFeed {
				return ""
			}
			var a *social.About
			datastore.DB().View(func(tx *bolt.Tx) error {
				a = social.GetAbout(tx, ref)
				return nil
			})
			buf := &bytes.Buffer{}
			err := ContentTemplates.ExecuteTemplate(buf, "avatar.tpl", struct {
				About *social.About
				Ref   ssb.Ref
			}{a, ref})
			if err != nil {
				log.Println(err)
			}
			return template.HTML(buf.String())
		},
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
		"GetVotes": func(ref ssb.Ref) (votes []*social.Vote) {
			datastore.DB().View(func(tx *bolt.Tx) error {
				votes = social.GetVotes(tx, ref)
				return nil
			})
			return
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
			return template.HTML("<!-- " + t + " --!>" + buf.String())
		},
	}).ParseGlob("templates/content/*.tpl"))
}

var PageTemplates = template.Must(template.New("index").Funcs(template.FuncMap{
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
		return template.HTML("<!-- " + t + " --!>" + buf.String())
	},
}).ParseGlob("templates/pages/*.tpl"))

func init() {
	log.Println(ContentTemplates.DefinedTemplates())
	log.Println(PageTemplates.DefinedTemplates())
}

func RegisterWebui() {
	bi := boltinspect.New(datastore.DB())

	http.HandleFunc("/bolt", bi.InspectEndpoint)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	http.HandleFunc("/", Index)
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/channel", Channel)
	http.HandleFunc("/post", Post)
	http.HandleFunc("/search", Search)
	http.HandleFunc("/publish/post", PublishPost)

	http.HandleFunc("/admin", Admin)
	http.HandleFunc("/rebuild", Rebuild)

	http.HandleFunc("/blob", Blob)
	http.HandleFunc("/blobinfo", BlobInfo)

	http.HandleFunc("/raw", Raw)

	http.HandleFunc("/upload", Upload)

	go http.ListenAndServe(":9823", nil)
}

func Upload(rw http.ResponseWriter, req *http.Request) {
	f, _, err := req.FormFile("upload")
	if err != nil {
		log.Println(err)
		PageTemplates.ExecuteTemplate(rw, "upload.tpl", nil)
		return
	}
	buf, _ := ioutil.ReadAll(f)
	bs := datastore.ExtraData("blobStore").(*blobs.BlobStore)
	ref := bs.Add(buf)
	http.Redirect(rw, req, "/blobinfo?id="+url.QueryEscape(ref.String()), http.StatusFound)
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
	err := PageTemplates.ExecuteTemplate(rw, "admin.tpl", struct {
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
	messages := datastore.LatestCountFiltered(100, graph.GetFollows(datastore, datastore.PrimaryRef, 1))
	err := PageTemplates.ExecuteTemplate(rw, "index.tpl", struct {
		Messages []*ssb.SignedMessage
	}{
		messages,
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
	err := PageTemplates.ExecuteTemplate(rw, "post.tpl", struct {
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
	err := PageTemplates.ExecuteTemplate(rw, "channel.tpl", struct {
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
	if query[0] == '#' {
		http.Redirect(rw, req, "/channel?channel="+query[1:], http.StatusFound)
		return
	}

	messages := search.Search(datastore, query, 50)
	err := PageTemplates.ExecuteTemplate(rw, "search.tpl", struct {
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

func BlobInfo(rw http.ResponseWriter, req *http.Request) {
	id := req.FormValue("id")
	if id == "" {
		http.NotFound(rw, req)
		return
	}
	r := ssb.ParseRef(id)
	PageTemplates.ExecuteTemplate(rw, "blob.tpl", struct {
		ID ssb.Ref
	}{
		ID: r,
	})
}

func Raw(rw http.ResponseWriter, req *http.Request) {
	id := req.FormValue("id")
	if id == "" {
		http.NotFound(rw, req)
		return
	}
	r := ssb.ParseRef(id)
	m := datastore.Get(nil, r)
	if m != nil {
		buf := m.Encode()
		rw.Write(buf)
	}
}
