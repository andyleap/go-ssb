package ssb

import (
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

func RemoveUnsupported(e *encoding.Encoder) *encoding.Encoder {
	return &encoding.Encoder{Transformer: &errorHandler{e, errorToRemove}}
}

type errorHandler struct {
	*encoding.Encoder
	handler func(dst []byte, r rune, err repertoireError) (n int, ok bool)
}

// TODO: consider making this error public in some form.
type repertoireError interface {
	Replacement() byte
}

func (h errorHandler) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	nDst, nSrc, err = h.Transformer.Transform(dst, src, atEOF)
	for err != nil {
		rerr, ok := err.(repertoireError)
		if !ok {
			return nDst, nSrc, err
		}
		r, sz := utf8.DecodeRune(src[nSrc:])
		n, ok := h.handler(dst[nDst:], r, rerr)
		if !ok {
			return nDst, nSrc, transform.ErrShortDst
		}
		err = nil
		nDst += n
		if nSrc += sz; nSrc < len(src) {
			var dn, sn int
			dn, sn, err = h.Transformer.Transform(dst[nDst:], src[nSrc:], atEOF)
			nDst += dn
			nSrc += sn
		}
	}
	return nDst, nSrc, err
}

func errorToRemove(dst []byte, r rune, err repertoireError) (n int, ok bool) {
	if len(dst) < 1 {
		return 0, false
	}
	dst[0] = byte(r)
	return 1, true
}

func ToJSBinary(src []byte) []byte {
	runes := []rune(string(src))
	utf := utf16.Encode(runes)
	out := make([]byte, len(utf))
	for i, r := range utf {
		out[i] = byte(r)
	}
	return out
}
