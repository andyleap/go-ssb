<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
<form action="/publish/post" method="post">
<textarea name="text" class="form-control"></textarea><br>
<input type="hidden" name="channel" value="{{.Channel}}">
<input type="hidden" name="returnto" value="/channel?channel={{.Channel}}">
<input type="submit" value="Publish!" class="btn btn-default">
</form>
</div>

{{range .Messages}}
<div class="well">
{{RenderContent . 1}}
</div>
{{end}}
</div>
</body>
</html>