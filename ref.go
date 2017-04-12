package ssb

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/ed25519"
)

type Ref string

type RefType int

func (rt RefType) String() string {
	switch rt {
	case RefFeed:
		return "@"
	case RefMessage:
		return "%"
	case RefBlob:
		return "&"
	default:
		return "?"
	}
}

const (
	RefInvalid RefType = iota
	RefFeed
	RefMessage
	RefBlob
)

type RefAlgo int

func (ra RefAlgo) String() string {
	switch ra {
	case RefAlgoSha256:
		return "sha256"
	case RefAlgoEd25519:
		return "ed25519"
	default:
		return "???"
	}
}

const (
	RefAlgoInvalid RefAlgo = iota
	RefAlgoSha256
	RefAlgoEd25519
)

var (
	ErrInvalidRefAlgo = errors.New("Invalid Ref Algo")
	ErrInvalidSig     = errors.New("Invalid Signature")
	ErrInvalidHash    = errors.New("Invalid Hash")
)

func (r Ref) Type() RefType {
	switch r[0] {
	case '@':
		return RefFeed
	case '%':
		return RefMessage
	case '&':
		return RefBlob
	}
	return RefInvalid
}

func (r Ref) Algo() RefAlgo {
	parts := strings.Split(string(r), ".")
	if len(parts) != 2 {
		return RefAlgoInvalid
	}
	switch strings.ToLower(parts[1]) {
	case "ed25519":
		return RefAlgoEd25519
	case "sha256":
		return RefAlgoSha256
	}
	return RefAlgoInvalid
}

func (r Ref) Raw() []byte {
	b64 := strings.Split(strings.TrimLeft(string(r), "@%&"), ".")[0]
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil
	}

	return raw
}

func (r Ref) CheckHash(content []byte) error {
	switch r.Algo() {
	case RefAlgoSha256:
		contentHash := sha256.Sum256(content)
		if bytes.Equal(r.Raw(), contentHash[:]) {
			return nil
		}
		return ErrInvalidHash
	}
	return ErrInvalidHash
}

type Signature string

type SigAlgo int

const (
	SigAlgoInvalid SigAlgo = iota
	SigAlgoEd25519
)

func (s Signature) Algo() SigAlgo {
	parts := strings.Split(string(s), ".")
	if len(parts) != 3 || parts[1] != "sig" {
		return SigAlgoInvalid
	}
	switch strings.ToLower(parts[2]) {
	case "ed25519":
		return SigAlgoEd25519
	}
	return SigAlgoInvalid
}

func (s Signature) Raw() []byte {
	b64 := strings.Split(string(s), ".")[0]
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil
	}

	return raw
}

func (s Signature) Verify(content []byte, r Ref) error {
	switch s.Algo() {
	case SigAlgoEd25519:
		if r.Algo() != RefAlgoEd25519 {
			return ErrInvalidSig
		}
		rawkey := r.Raw()
		if rawkey == nil {
			return nil
		}

		key := ed25519.PublicKey(rawkey)
		if ed25519.Verify(key, content, s.Raw()) {
			return nil
		}
		return ErrInvalidSig
	}
	return ErrInvalidSig
}
