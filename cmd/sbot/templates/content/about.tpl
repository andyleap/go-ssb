{{Avatar .Message.Author}}
{{if eq .Message.Author.String .Content.About.String}}
self identifies as {{.Content.Name}}
{{else}}
identifies <a href="/feed?id={{urlquery .Content.About}}">{{.Content.About}}</a> as {{.Content.Name}}
{{end}}