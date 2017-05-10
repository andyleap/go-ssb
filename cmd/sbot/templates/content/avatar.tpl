{{if .About}}
{{if ne .About.Image.Link.String ""}}<img src="/blob?id={{urlquery .About.Image.Link}}" width="100" height="100"><br>{{end}}
<b>{{.About.Name}}</b>
{{else}}
{{.Ref}}
{{end}}