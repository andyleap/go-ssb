<a href="/feed?id={{urlquery .Ref}}">{{if .About}}
{{if .About.Image}}<img class="avatar" src="/blob?id={{urlquery .About.Image.Link}}"><br>{{end}}
<b>{{.About.Name}}</b>
{{else}}
{{.Ref}}
{{end}}</a>
