<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">
{{template "navbar.tpl"}}

<div class="row">
<div class="col-sm-6 col-sm-offset-2">
{{if .Profile}}
{{if .Profile.Image}}<img src="/blob?id={{urlquery .Profile.Image.Link}}" width="100" height="100"><br>{{end}}
<b>{{.Profile.Name}}</b>
{{else}}
{{.Ref}}
{{end}}
</div>
<div class="col-sm-2">
<form action="/publish/follow" method="post">
<input type="hidden" name="feed" value="{{.Ref}}">
<input type="hidden" name="returnto" value="/feed?id={{urlquery .Ref}}">
<button class="btn btn-default" style="float:right;">Follow</button>
</form>

</div>
</div>

{{range .Messages}}
<div class="well">
	{{RenderContent . 1}}
</div>
{{end}}
</div>

</div>
</body>
</html>