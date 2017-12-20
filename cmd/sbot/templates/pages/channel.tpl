<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
<p class="channel">
<a class="channel" href="/channel?channel={{.Channel}}">#{{.Channel}}</a>
</p>
<form action="/publish/post" method="post">
<textarea name="text" class="form-control"></textarea><br>
<input type="hidden" name="channel" value="{{.Channel}}">
<input type="hidden" name="returnto" value="/channel?channel={{.Channel}}">
<input type="submit" value="Publish!" class="btn btn-primary">
</form>
</div>

{{range .Messages}}
<div class="well">
{{RenderContent . 1}}
</div>
{{end}}


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

</div>
</body>
</html>
