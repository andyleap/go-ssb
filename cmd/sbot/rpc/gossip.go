package rpc

type AddPubReq struct {
	Host   string
	Port   int
	PubKey string
}

type AddPubRes struct {
	Err error
}
