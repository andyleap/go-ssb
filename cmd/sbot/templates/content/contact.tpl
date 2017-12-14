{{Avatar .Message.Author}}{{if .Content.Following}} followed {{else}} unfollowed {{end}}
{{$contact := GetAbout .Content.Contact}}
<a href="/feed?id={{urlquery .Content.Contact}}">
{{if $contact}}
{{$contact.Name}}
{{else}}
{{.Content.Contact}}
{{end}}
</a>
<a href="/post?id={{.Message.Key}}">{{RenderJSTime .Message.Timestamp}}</a>
{{$votes := GetVotes .Message.Key}}{{len $votes}} Votes
