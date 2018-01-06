<html>
<head>
{{template "header.tpl"}}
</head>
<body>
<base target="_parent">
<div class="thread">
<div class="container">

{{RenderContent .Root 0}}
{{range .Messages}}
{{RenderContent . 0}}
{{end}}


<div class="postblock">
<div class="facebox">
{{if .Profile}}
{{if .Profile.Image}}<img class="avatar" src="/blob?id={{urlquery .Profile.Image.Link}}"><br>{{end}}
<b>{{.Profile.Name}}</b>
{{else}}
{{.Ref}}
{{end}}
</div>
<form action="/publish/post" method="post">
<div class="coolpost" style="padding:0;">
<textarea class="poster" name="text" class="form-control"></textarea><br>
<input type="hidden" name="channel" value="{{.Channel}}">
<input type="hidden" name="returnto" value="/thread?id={{urlquery .Root.Key.String}}">
<input type="hidden" name="branch" value="{{.Reply.String}}">
<input type="hidden" name="root" value="{{.Root.Key.String}}">
</div>
<input type="submit" value="Publish!" class="postbutton">
</form>
</div>
</div>

<a class="mylink" href="/">Go home</a>

</div>
</div>
</body>
</html>
