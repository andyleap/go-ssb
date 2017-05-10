<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
{{.ID}}<br>
<img src="/blob?id={{urlquery .ID}}">
</div>
</div>
</body>
</html>