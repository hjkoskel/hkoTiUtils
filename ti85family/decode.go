package ti85family

import (
	"fmt"
)

// Token is one decoded program token.
type Token struct {
	Offset int    // byte offset in the token stream
	Code   uint16 // 1-byte code, or prefix<<8|byte for 0x3D/0x8E tokens
	Arg    []byte // payload: string, number, name, label, from-unit, asm bytes
	Arg2   []byte // to-unit of a conversion token
	Cache  [2]byte
	// Open is set when a zero-terminated payload ran to the end of the data
	// without its terminator.
	Open bool
	// Unknown is set for codes missing from the token table.
	Unknown bool
}

// Decode splits a raw token stream into tokens. It never fails on unknown
// codes (they get Unknown set). An error is returned only for payloads that
// are cut short; the tokens decoded so far are returned with it.
func (dl *Dialect) Decode(b []byte) ([]Token, error) {
	var out []Token
	for i := 0; i < len(b); {
		t := Token{Offset: i, Code: uint16(b[i])}
		c := b[i]
		i++
		switch {
		case c == tokString || c == tokNumber || c == tokLbl || c == tokGoto || c == tokRemark:
			if c == tokGoto {
				if len(b)-i < 2 {
					t.Arg = b[i:]
					return append(out, t), fmt.Errorf("offset %d: truncated Goto", t.Offset)
				}
				copy(t.Cache[:], b[i:i+2])
				i += 2
			}
			end := i
			for end < len(b) && b[end] != 0 {
				end++
			}
			t.Arg = b[i:end]
			if end == len(b) {
				t.Open = true
				i = end
			} else {
				i = end + 1
			}
		case c == tokNameQ || c == tokNameY || c == tokNameStat:
			if i >= len(b) || i+1+int(b[i]) > len(b) {
				t.Arg = b[i:]
				return append(out, t), fmt.Errorf("offset %d: truncated name", t.Offset)
			}
			n := int(b[i])
			t.Arg = b[i+1 : i+1+n]
			i += 1 + n
		case c >= tokName1 && c <= tokName8:
			n := int(c-tokName1) + 1
			if i+n > len(b) {
				t.Arg = b[i:]
				return append(out, t), fmt.Errorf("offset %d: truncated name", t.Offset)
			}
			t.Arg = b[i : i+n]
			i += n
		case c == tokConvert:
			if i >= len(b) || i+1+int(b[i]) >= len(b) {
				return append(out, t), fmt.Errorf("offset %d: truncated conversion", t.Offset)
			}
			n := int(b[i])
			t.Arg = b[i+1 : i+1+n]
			i += 1 + n
			m := int(b[i])
			if i+1+m > len(b) {
				return append(out, t), fmt.Errorf("offset %d: truncated conversion", t.Offset)
			}
			t.Arg2 = b[i+1 : i+1+m]
			i += 1 + m
		case c == tokPrefix3D || c == tokPrefix8E:
			if i >= len(b) {
				t.Unknown = true // lone prefix byte
				break
			}
			t.Code = uint16(c)<<8 | uint16(b[i])
			i++
			switch {
			case t.Code == tokAsmPrgm || t.Code == tokAsmCompile:
				t.Arg = b[i:]
				i = len(b)
			case t.Code == tokExtCall:
				if i+3 > len(b) || i+3+int(b[i+2]) > len(b) {
					t.Arg = b[i:]
					return append(out, t), fmt.Errorf("offset %d: truncated 8E32 call", t.Offset)
				}
				copy(t.Cache[:], b[i:i+2])
				n := int(b[i+2])
				t.Arg = b[i+3 : i+3+n]
				i += 3 + n
			case t.Code == tokEditLock:
			default:
				if _, ok := dl.byCode[t.Code]; !ok {
					t.Unknown = true
				}
			}
		default:
			if _, ok := dl.byCode[t.Code]; !ok {
				t.Unknown = true
			}
		}
		out = append(out, t)
	}
	return out, nil
}
