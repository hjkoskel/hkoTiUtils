package ti85family

type payloadKind int

const (
	pkNone     payloadKind = iota
	pkZeroTerm             // string, number, Lbl, remark: arg + 00
	pkGoto                 // cache(2) + arg + 00
	pkLenName              // len + arg
	pkFixName              // arg (length given by the code)
	pkConvert              // len + arg + len + arg2
	pkExtCall              // cache(2) + len + arg
	pkRest                 // arg runs to the end of the program
)

func kindOf(code uint16) payloadKind {
	switch {
	case code == tokString || code == tokNumber || code == tokLbl || code == tokRemark:
		return pkZeroTerm
	case code == tokGoto:
		return pkGoto
	case code == tokNameQ || code == tokNameY || code == tokNameStat:
		return pkLenName
	case code >= tokName1 && code <= tokName8:
		return pkFixName
	case code == tokConvert:
		return pkConvert
	case code == tokExtCall:
		return pkExtCall
	case code == tokAsmPrgm || code == tokAsmCompile:
		return pkRest
	}
	return pkNone
}

// AppendBytes appends the encoded token to dst.
func (t *Token) AppendBytes(dst []byte) []byte {
	if t.Code == tokRaw {
		return append(dst, t.Arg...)
	}
	if t.Code > 0xFF {
		dst = append(dst, byte(t.Code>>8), byte(t.Code))
	} else {
		dst = append(dst, byte(t.Code))
	}
	switch kindOf(t.Code) {
	case pkZeroTerm, pkGoto:
		if t.Code == tokGoto {
			dst = append(dst, t.Cache[:]...)
		}
		dst = append(dst, t.Arg...)
		if !t.Open {
			dst = append(dst, 0)
		}
	case pkLenName:
		dst = append(dst, byte(len(t.Arg)))
		dst = append(dst, t.Arg...)
	case pkFixName, pkRest:
		dst = append(dst, t.Arg...)
	case pkConvert:
		dst = append(dst, byte(len(t.Arg)))
		dst = append(dst, t.Arg...)
		dst = append(dst, byte(len(t.Arg2)))
		dst = append(dst, t.Arg2...)
	case pkExtCall:
		dst = append(dst, t.Cache[:]...)
		dst = append(dst, byte(len(t.Arg)))
		dst = append(dst, t.Arg...)
	}
	return dst
}

// Encode turns tokens back into a raw token stream.
func Encode(toks []Token) []byte {
	var out []byte
	for i := range toks {
		out = toks[i].AppendBytes(out)
	}
	return out
}
