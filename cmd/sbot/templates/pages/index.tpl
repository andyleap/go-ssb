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

<div class="pagnum">
<div class="page-nav">
{{if not (eq .PageStr "1")}}<form .class="nav" action="/?page={{.PrevPage}}" method="post">
<button>less</button>
</form>
{{else}}
{{end}}
<form .class="nav" action="/?page={{.NextPage}}" method="post">
<button>more</button>
</form>
</div>
</div>

</body></html>
