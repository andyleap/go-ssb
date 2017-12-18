<html>
<head>
{{template "header.tpl"}}
</head>
<body>

{{template "navbar.tpl"}}


<form class="postingarea" action="/publish/post" method="post">
<textarea name="text"></textarea><br>
<input type="hidden" name="returnto" value="/">
<input type="submit" value="Publish!!" class="btn btn-primary">
</form>

<br>

{{range .Messages}}
{{RenderContent . 1}}
{{end}}


<div class="bigbutton">
<a class="bigbutton" href="#addingpub">
+ pub
</a>
</div>

<a href="#main" class="pubwindow" id="addingpub">
<div class="exit">&#10005</div>
<iframe src="/addpub" class="pubforward" id="floater">
Your browser doesn't support iframes
</iframe>
</a>


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
