<html>
<head>
{{template "header.tpl"}}
</head>
<body>
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

</body>
</html>
