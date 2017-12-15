{{if ge .Levels 0}}
{{if .Content.Branch.IsMessage}}
<a href="/thread?id={{.Content.Root}}">View full thread</a><br>
{{$prior := GetMessage .Content.Branch}}{{RenderContent $prior .Levels}}
{{end}}
{{end}}
{{Avatar .Message.Author}}
<a href="/post?id={{.Message.Key}}">{{RenderJSTime .Message.Timestamp}}</a><br>
{{if ne .Content.Channel ""}}<a href="/channel?channel={{urlquery .Content.Channel}}">#{{.Content.Channel}}</a>{{end}}
{{$votes := GetVotes .Message.Key}}{{len $votes}} Votes
{{Markdown .Content.Text}}
