<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
<a href="/repo/want?id={{urlquery .Ref}}" class="btn btn-primary">Want all blobs</a>
</div>

<div class="well">
{{range .Issues}}
<div class="well">
{{RenderContent . 1}}
</div>
{{end}}
</div>

<div class="well">
{{range .Blobs}}
<a href="/blob?id={{urlquery .}}">{{.}}</a> - {{if HasBlob .}}<span style="color: green;">Present</span>{{else}}<span style="color: red;">Not Present</span>{{end}}<br>
{{end}}
</div>

<div class="well">
{{range .Updates}}
<a href="/raw?id={{urlquery .}}">{{.}}</a><br>
{{end}}
</div>
</div>
</body>
</html>