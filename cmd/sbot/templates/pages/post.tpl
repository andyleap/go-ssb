<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
{{RenderContent .Message 1}}
<div class="container-fluid vote-container">
{{range .Votes}}
{{$vote := Decode .}}
<div class="vote {{if gt $vote.Vote.Value 0}}vote-up{{else if lt $vote.Vote.Value 0}}vote-down{{end}}">
{{RenderContentTemplate . 0 "vote-simple"}}
</div>
{{end}}
</div>
</div>

<div class="well">
<form action="/publish/post" method="post">
<textarea name="text" class="form-control"></textarea><br>
<input type="hidden" name="channel" value="{{.Content.Channel}}">
<input type="hidden" name="branch" value="{{.Message.Key}}">
<input type="hidden" name="root" value="{{if eq .Content.Root.Type 0}}{{.Message.Key}}{{else}}{{.Content.Root}}{{end}}">
<input type="hidden" name="returnto" value="/post?id={{.Message.Key | urlquery}}">
<input type="submit" value="Publish!" class="btn btn-default">
</form>
</div>

</div>
</body>
</html>