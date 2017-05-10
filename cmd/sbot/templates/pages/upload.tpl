<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<div class="container">

{{template "navbar.tpl"}}

<div class="well">
<form action="/upload" method="post" enctype="multipart/form-data">
<input type="file" name="upload">
<input type="submit" value="Upload!" class="btn btn-default">
</form>
</div>
</div>
</body>
</html>