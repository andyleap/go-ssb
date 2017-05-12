{{Avatar .Message.Author}}
{{if .Content.Subscribed}}subscribed to{{else}}unsubscribed from{{end}} <a href="/channel?channel={{urlquery .Content.Channel}}">#{{.Content.Channel}}</a>