<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

{{range .Messages}}
<div class="well">
{{RenderContent .}}
</div>
{{end}}
</div>
</body>
</html>