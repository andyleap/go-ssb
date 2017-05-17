<div class="container-fluid"><div class="row">
<div class="col-sm-7">{{Avatar .Message.Author}} pushed {{len .Content.Commits}} commits</div>
<div class="col-sm-5">
<div class="row">
<div class="col-xs-8">
<a href="/post?id={{.Message.Key}}">{{RenderJSTime .Message.Timestamp}}</a>
</div>
<div class="col-xs-4" style="text-align: right;">
{{$votes := GetVotes .Message.Key}}{{len $votes}} Votes
</div>
</div>
{{if .Content.Repo.IsMessage}}<a href="/repo?id={{urlquery .Content.Repo}}">{{.Content.Repo}}</a>{{end}}

</div>
</div></div>
<div>

{{range .Content.Commits}}
<hr>
Commit {{.Sha1}}<br>
<b>{{.Title}}</b><br>
{{.Body}}
{{end}}
</div>