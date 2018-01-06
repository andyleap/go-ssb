<html>
<head>
{{template "header.tpl"}}
</head>
<body>

{{template "navbar.tpl"}}


<div class="main">
{{range .Messages}}
{{RenderContent . 1}}
{{end}}
</div>

<div class="page-nav">
{{if not (eq .PageStr "1")}}<form action="/?page={{.PrevPage}}" method="post">
<button class="mylink">less</button>
</form>
{{else}}
{{end}}
<form action="/?page={{.NextPage}}" method="post">
<button class="mylink">more</button>
</form>
</div>
</div>

</body></html>
