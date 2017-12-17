<a href="/feed?id={{urlquery .Ref}}">{{if .About}}
{{if .About.Image}}<img class="logo" src="/static/logo.svg"><br>{{end}}
<div class="pname">{{.About.Name}}</div>
{{else}}
{{.Ref}}
{{end}}</a>
