<div class="postblock">
<div class="facebox">
{{Avatar .Message.Author}}
<br>
<time><a href="/thread?id={{.Message.Key}}">{{RenderJSTime .Message.Timestamp}}</a><br></time>

{{if ne .Content.Channel ""}}<a href="/channel?channel={{urlquery .Content.Channel}}">#{{.Content.Channel}}</a>{{end}}

</div>

<div class="coolpost">

<div class="votes">
<a class="voteman" href="/post?id={{.Message.Key}}">{{$votes := GetVotes .Message.Key}}{{len $votes}} Votes
<div class="hidebox">

<!--figure out how to put voters here-->

</div></a>

<form class="votebutton" action="/publish/vote" method="post">
<input type="hidden" name="link" value="{{.Message.Key}}">
<input type="hidden" name="returnto" value="/post?id={{.Message.Key | urlquery}}">
<input type="submit" value="+" class="voter">
</form>
</div>


<div class="goofyBS">
{{if ge .Levels 0}}
{{if .Content.Branch.IsMessage}}

<a href="#{{.Content.Root}}">
View full thread
</a>

<a href="#main" class="mylightbox" id="{{.Content.Root}}">
<div class="exit">&#10005</div>
<iframe src="/thread?id={{.Content.Root}}" class="mythread" id="mypost">
Your browser doesn't support iframes
</iframe>
</a>

{{$prior := GetMessage .Content.Branch}}{{RenderContent $prior .Levels}}
{{end}}
{{end}}
</div>
{{Markdown .Content.Text}}


<a href="/raw?id={{.Message.Key}}" class="rawbutton">json</a>


</div>

</div>
