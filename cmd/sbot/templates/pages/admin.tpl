<html>
<head>
</head>
<body>
<table>
<tr><th>key</th><th>size</th></tr>
{{range $b, $size := .DBSize}}
<tr><td>{{$b}}</td><td style="text-align: right;">{{$size}}</td>
{{end}}
</table><br>
<a href="/rebuild?module=all">all</a><br>
{{range .Modules}}
<a href="/rebuild?module={{.}}">{{.}}</a><br>
{{end}}
</body>
</html>