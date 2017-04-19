package rpc

type PostReq struct {
	Feed    string
	Text    string
	Root    string
	Branch  string
	Channel string
}

type PostRes struct {
	Err error
}

type FollowReq struct {
	Feed    string
	Contact string
}

type FollowRes struct {
	Err error
}

type AboutReq struct {
	Feed string
	Name string
}

type AboutRes struct {
	Err error
}
