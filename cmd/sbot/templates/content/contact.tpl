{{Avatar .Message.Author}}
{{if .Content.Following}} followed {{else}} unfollowed {{end}}
{{$contact := GetAbout .Content.Contact}}
{{if $contact}}
{{$contact.Name}}
{{else}}
{{.Content.Contact}}
{{end}}