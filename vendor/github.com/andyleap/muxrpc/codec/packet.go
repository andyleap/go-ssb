/*
This file is part of go-muxrpc.

go-muxrpc is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

go-muxrpc is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with go-muxrpc.  If not, see <http://www.gnu.org/licenses/>.
*/

package codec

import (
	"encoding/json"
	"fmt"
)

// Packet is the decoded high-level representation
type Packet struct {
	Stream bool
	EndErr bool
	Type   PacketType
	Req    int32
	Body   []byte
}

func (p Packet) String() string {
	s := fmt.Sprintf("Stream(%v) EndErr(%v) ", p.Stream, p.EndErr)
	s += fmt.Sprintf("Type(%s) Len(%d) Req(%d)\n", p.Type.String(), len(p.Body), p.Req)
	if p.Type == JSON {
		var i interface{}
		if err := json.Unmarshal(p.Body, &i); err != nil {
			s += fmt.Sprintf("json.Unmarshal error: %s", err)
			return s
		}
		s += fmt.Sprintf("Body: %+v", i)
	} else {
		if len(p.Body) > 50 {
			s += fmt.Sprintf("%q...", p.Body[:50])
		} else {
			s += fmt.Sprintf("%q", p.Body)
		}
	}
	return s
}

// Flag is the first byte of the Header
type Flag byte

// Flag bitmasks
const (
	FlagString Flag = 1 << iota // type
	FlagJSON                    // bits
	FlagEndErr
	FlagStream
)

// PacketType are the 2 bits of type in the packet header
type PacketType uint

func (pt PacketType) Flag() Flag {
	switch pt {
	case String:
		return FlagString
	case JSON:
		return FlagJSON
	}
	return 0
}

func (f Flag) PacketType() PacketType {
	return PacketType((byte(f) & 3))
}

// Enumeration of the possible body types of a packet
const (
	Buffer PacketType = iota
	String
	JSON
)

// Header is the wire representation of a packet header
type Header struct {
	Flag Flag
	Len  uint32
	Req  int32
}
