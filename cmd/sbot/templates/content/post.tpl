{{$author := GetAbout .Message.Author}}<b>{{$author.Name}}</b><sub style="float: right;">{{RenderJSTime .Message.Timestamp}}</sub><br/>
{{Markdown .Content.Text}}