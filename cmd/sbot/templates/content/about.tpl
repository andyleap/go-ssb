{{Avatar .Message.Author}}
{{if eq .Message.Author .Content.Link}}
self identifies as {{.Content.Name}}
{{else}}
identifies {{.Content.Link}} as {{.Content.Name}}
{{end}}