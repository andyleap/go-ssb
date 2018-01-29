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
{{if .Profile.Image}}<img class="avatarwell" src="/blob?id={{urlquery .Profile.Image.Link}}">{{end}}
<div class="about"><b>@{{.Profile.Name}}</b><p class="ref">{{.Ref}}</p><form action="/publish/follow" method="post">
<input type="hidden" name="feed" value="{{.Ref}}">
<input type="hidden" name="returnto" value="/feed?id={{urlquery .Ref}}">
<button class="postbutton" style="float:right;">Follow</button>
</form>
</div>
{{else}}
{{.Ref}}
{{end}}
<br>

<div>
<form action="/publish/about" method="post" enctype="multipart/form-data" class="form-inline">
<input type="text" name="name" class="form-control" placeholder="name">
<input type="file" name="upload" class="form-control" placeholder="picture">
<input type="submit" value="Update" class="btn btn-primary">
</form>
</div>
</div>

<p>
{{range $k, $v := .Follows}}
<a href="/feed?id={{$k}}">{{Avatar $k}}</a>
{{end}}
</p>

<br>
</div>

{{range .Messages}}
<div class="well">
	{{RenderContent . 1}}
</div>
{{end}}
</div>

<div class="pagnum">
<div class="page-nav">
{{if not (eq .PageStr "1")}}<form .class="nav" action="/feed?id={{.Ref}}&page={{.PrevPage}}" method="post">
<button>less</button>
</form>
{{else}}
{{end}}
<form .class="nav" action="/feed?id={{.Ref}}&page={{.NextPage}}" method="post">
<button>more</button>
</form>
</div>
</div>

</div>
</body>
</html>
