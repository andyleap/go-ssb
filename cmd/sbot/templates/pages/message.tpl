<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
{{RenderContent .Message 1}}
</div>

<div class="well">
<form action="/publish/post" method="post">
<textarea name="text" class="form-control"></textarea><br>
<input type="hidden" name="branch" value="{{.Message.Key}}">
<input type="hidden" name="root" value="{{.Message.Key}}">
<input type="hidden" name="returnto" value="/post?id={{urlquery .Message.Key}}">
<input type="submit" value="Publish!" class="btn btn-primary">
</form>
</div>

</div>
</body>
</html>
