package ssb

import (
	"encoding/base64"

	"golang.org/x/crypto/ed25519"
)

type Signer interface {
	Sign([]byte) Signature
}

type SignerEd25519 struct {
	Private ed25519.PrivateKey
}

func (k SignerEd25519) Sign(content []byte) Signature {
	return Signature(base64.StdEncoding.EncodeToString(ed25519.Sign(k.Private, content)) + ".sig.ed25519")
}
