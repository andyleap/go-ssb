<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
{{if .Profile}}
{{if .Profile.Image}}<img class="avatarwell" src="/blob?id={{urlquery .Profile.Image.Link}}"><br>{{end}}
</div>
<div>
<b>{{.Profile.Name}}</b>
{{else}}
{{.Ref}}
{{end}}
{{.Ref}}
</div>

<div>
<form action="/publish/about" method="post" enctype="multipart/form-data" class="form-inline">
<input type="text" name="name" class="form-control" placeholder="name">
<input type="file" name="upload" class="form-control" placeholder="picture">
<input type="submit" value="Update" class="btn btn-primary">
</form>
</div>
</div>

<div>
{{range .Messages}}
{{RenderContent . 1}}
{{end}}
</div>

</body>
</html>
