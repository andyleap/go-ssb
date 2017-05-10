{{Avatar .Message.Author}}
{{if ne .Content.Vote.Reason ""}}{{.Content.Vote.Reason}}{{else}}
{{if gt .Content.Vote.Value 0}}liked
{{else if lt .Content.Vote.Value 0}}disliked
{{else}}noted{{end}}{{end}}
 <a href="/post?id={{.Content.Vote.Link}}">{{.Content.Vote.Link}}</a>