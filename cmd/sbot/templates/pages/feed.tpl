<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

{{range .Messages}}
<div class="well">
{{RenderContent . 1}}
</div>
{{end}}
</div>
</body>
</html>