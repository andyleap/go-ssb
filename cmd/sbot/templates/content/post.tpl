{{if .Content.Branch.IsMessage}}
<div style="color:grey; border-left: 1px solid black; padding-left: 20px;">
{{$prior := GetMessage .Content.Branch}}{{RenderContent $prior}}
</div>
{{end}}
{{$author := GetAbout .Message.Author}}<b>{{$author.Name}}</b><sub style="float: right;"><a href="/post?id={{.Message.Key}}">{{RenderJSTime .Message.Timestamp}}</a></sub><br/>
{{Markdown .Content.Text}}