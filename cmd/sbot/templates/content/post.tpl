{{if not .Embedded}}
{{if .Content.Branch.IsMessage}}
<a href="/post?id={{.Content.Root}}">View full thread</a><br>
<div style="color:grey; border-left: 1px solid black; padding-left: 20px;">
{{$prior := GetMessage .Content.Branch}}{{RenderContent $prior}}
</div>
{{end}}
{{end}}
<sub style="float: right;"><a href="/post?id={{.Message.Key}}">{{RenderJSTime .Message.Timestamp}}<br></a>
{{if ne .Content.Channel ""}}#{{.Content.Channel}}{{end}}
{{$votes := GetVotes .Message.Key}}<span style="float: right;">{{len $votes}} Votes</span>
</sub>
{{Avatar .Message.Author}}
{{Markdown .Content.Text}}