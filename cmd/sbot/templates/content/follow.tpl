<a href="/feed?id={{urlquery .Ref}}">{{if .About}}
{{if .About.Image}}<img alt="{{.About.Name}}" width="30" height="30" src="/blob?id={{urlquery .About.Image.Link}}">{{end}}
{{else}}
{{end}}</a>
