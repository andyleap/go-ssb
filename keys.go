package ssb

import "golang.org/x/crypto/ed25519"

type Key struct {
	Curve  string
	ID     string
	Signer `json:"-"`
}

type Signer interface {
	Sign([]byte) []byte
}

type SignerEd25519 struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
}

func (k SignerEd25519) Sign(content []byte) []byte {
	return ed25519.Sign(k.Private, content)
}
