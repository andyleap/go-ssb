{{if ge .Levels 0}}
{{if .Content.Branch.IsMessage}}
<a href="/thread?id={{.Content.Root}}">View full thread</a><br>
<div class="well">
{{$prior := GetMessage .Content.Branch}}{{RenderContent $prior .Levels}}
</div>
{{end}}
{{end}}
<div class="container-fluid"><div class="row">
<div class="col-sm-7">{{Avatar .Message.Author}}</div>
<div class="col-sm-5">
<a href="/post?id={{.Message.Key}}">{{RenderJSTime .Message.Timestamp}}</a><br>
<div class="row">
<div class="col-xs-8">
{{if ne .Content.Channel ""}}<a href="/channel?channel={{urlquery .Content.Channel}}">#{{.Content.Channel}}</a>{{end}}
</div>
<div class="col-xs-4" style="text-align: right;">
{{$votes := GetVotes .Message.Key}}{{len $votes}} Votes
</div>
</div>
</div>
</div></div>
<div>
{{Markdown .Content.Text}}
</div>