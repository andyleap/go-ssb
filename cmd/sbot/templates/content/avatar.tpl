<a href="/feed?id={{urlquery .Ref}}">{{if .About}}
{{if .About.Image}}<img class="logo" src="/static/logo.svg"><br>{{end}}
<b>{{.About.Name}}</b>
{{else}}
{{.Ref}}
{{end}}</a>
