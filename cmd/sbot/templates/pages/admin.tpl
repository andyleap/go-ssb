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
<a class="btn btn-default" href="/rebuild?module=all">all</a>
{{range .Modules}}
<a class="btn btn-default" href="/rebuild?module={{.}}">{{.}}</a>
{{end}}

<div class="well">
<form action="/gossip/add" method="post">
<div class="form-group">
<input type="text" name="host" class="form-control" placeholder="Host">
<input type="text" name="port" class="form-control" placeholder="Port">
<input type="text" name="key" class="form-control" placeholder="Key">
<input type="submit" value="Add Peer" class="btn btn-primary">
</div>
</form>
</div>

<div class="well">
<form action="/gossip/accept" method="post">
<div class="form-group">
<input type="text" name="invite" class="form-control" placeholder="Invite">
<div class="checkbox">
  <label>
    <input type="checkbox" name="follow" value="follow" checked="checked">
    Automatically follow the pub
  </label>
</div>
<input type="submit" value="Accept" class="btn btn-primary">
</div>
</form>
</div>

</div>
</body>
</html>