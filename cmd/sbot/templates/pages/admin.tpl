<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">
{{template "navbar.tpl"}}
<table class="table table-striped table-bordered table-hover">
<tr><th>key</th><th>size</th></tr>
{{range $b, $size := .DBSize}}
<tr><td>{{$b}}</td><td style="text-align: right;">{{$size}}</td>
{{end}}
</table><br>
<a href="/rebuild?module=all">all</a><br>
{{range .Modules}}
<a href="/rebuild?module={{.}}">{{.}}</a><br>
{{end}}
</div>
</body>
</html>