<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
<form action="/upload" method="post" enctype="multipart/form-data" class="form-inline">
<input type="file" name="upload" class="form-control">
<input type="submit" value="Upload!" class="btn btn-primary">
</form>
</div>
</div>
</body>
</html>