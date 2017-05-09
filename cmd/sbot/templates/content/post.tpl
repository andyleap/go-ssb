{{if not .Embedded}}
{{if .Content.Branch.IsMessage}}
<a href="/post?id={{.Content.Root}}">View full thread</a><br>
<div style="color:grey; border-left: 1px solid black; padding-left: 20px;">
{{$prior := GetMessage .Content.Branch}}{{RenderContent $prior}}
</div>
{{end}}
{{end}}
{{$author := GetAbout .Message.Author}}{{if $author}}
{{if ne $author.Image.Link.String ""}}<img src="/blob?id={{urlquery $author.Image.Link}}" width="100" height="100">{{end}}
<b>{{$author.Name}}</b>
{{else}}
{{.Message.Author}}
{{end}}
<sub style="float: right;"><a href="/post?id={{.Message.Key}}">{{RenderJSTime .Message.Timestamp}}</a></sub><br/>
{{Markdown .Content.Text}}