{{if eq .Message.Author .Content.Link}}
{{if .Content.Name}}{{.Content.Link}} identifies as {{.Content.Name}}{{end}}
{{else}}
{{if .Content.Name}}{{.Message.Author}} identifies {{.Content.Link}} as {{.Content.Name}}{{end}}
{{end}}