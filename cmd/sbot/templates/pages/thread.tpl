<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
{{RenderContent .Root 0}}
</div>
{{range .Messages}}
<div class="well">
{{RenderContent . 0}}
</div>
{{end}}

<div class="well">
<form action="/publish/post" method="post">
<textarea name="text" class="form-control"></textarea><br>
<input type="hidden" name="channel" value="{{urlquery .Channel}}">
<input type="hidden" name="returnto" value="/feed?id={{urlquery .Root.Key.String}}">
<input type="hidden" name="branch" value="{{urlquery .Reply.String}}">
<input type="hidden" name="root" value="{{urlquery .Root.Key.String}}">
<input type="submit" value="Publish!" class="btn btn-default">
</form>
</div>

</div>
</body>
</html>