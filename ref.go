package ssb

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/ed25519"
)

type Ref struct {
	Type RefType
	Data string
	Algo RefAlgo
}

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
	ErrInvalidRefType = errors.New("Invalid Ref Type")
	ErrInvalidRefAlgo = errors.New("Invalid Ref Algo")
	ErrInvalidSig     = errors.New("Invalid Signature")
	ErrInvalidHash    = errors.New("Invalid Hash")
)

func NewRef(typ RefType, raw []byte, algo RefAlgo) (Ref, error) {
	return Ref{typ, string(raw), algo}, nil
}

func (r Ref) Raw() []byte {
	return []byte(r.Data)
}

func (r Ref) DBKey() []byte {
	return append([]byte{byte(r.Type), byte(r.Algo)}, []byte(r.Data)...)
}

func DBRef(ref []byte) Ref {
	return Ref{Type: RefType(ref[0]), Data: string(ref[2:]), Algo: RefAlgo(ref[1])}
}

func ParseRef(ref string) Ref {
	parts := strings.Split(strings.Trim(ref, "@%&"), ".")
	if len(parts) != 2 {
		return Ref{}
	}
	r := Ref{}
	switch ref[0] {
	case '@':
		r.Type = RefFeed
	case '%':
		r.Type = RefMessage
	case '&':
		r.Type = RefBlob
	default:
		return Ref{}
	}
	switch strings.ToLower(parts[1]) {
	case "sha256":
		r.Algo = RefAlgoSha256
	case "ed25519":
		r.Algo = RefAlgoEd25519
	default:
		return Ref{}
	}
	buf, _ := base64.StdEncoding.DecodeString(parts[0])
	r.Data = string(buf)
	return r
}

func (r Ref) String() string {
	return r.Type.String() + base64.StdEncoding.EncodeToString([]byte(r.Data)) + "." + r.Algo.String()
}

func (r Ref) MarshalText() (text []byte, err error) {
	return []byte(r.String()), nil
}

func (r *Ref) UnmarshalText(text []byte) error {
	*r = ParseRef(string(text))
	return nil
}

func (r Ref) CheckHash(content []byte) error {
	switch r.Algo {
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
		if r.Algo != RefAlgoEd25519 {
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
