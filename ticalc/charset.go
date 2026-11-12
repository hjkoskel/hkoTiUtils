package ticalc

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Escape brackets used for bytes/tokens that have no text form. A raw byte is
// written as ⟦NN⟧ and a two-byte token as ⟦NNNN⟧ (upper-case hex). Both
// Charset.Encode and the tokenizers read them back, so text output always
// round-trips to the same bytes.
const (
	EscOpen  = '⟦'
	EscClose = '⟧'
)

// Charset maps calculator font bytes to Unicode text and back.
type Charset struct {
	toText [256]string
	toByte map[string]byte
	maxLen int // longest text in runes (for greedy encoding)
}

// NewCharset builds a charset. Bytes missing from m (and not printable ASCII
// when asciiDefault is set) are written as escapes.
func NewCharset(asciiDefault bool, m map[byte]string) *Charset {
	c := &Charset{toByte: map[string]byte{}}
	if asciiDefault {
		for b := 0x20; b < 0x7F; b++ {
			c.toText[b] = string(rune(b))
		}
	}
	for b, s := range m {
		c.toText[b] = s
	}
	for b := 0; b < 256; b++ {
		s := c.toText[b]
		if s == "" {
			continue
		}
		if _, dup := c.toByte[s]; !dup {
			c.toByte[s] = byte(b)
		}
		if n := utf8.RuneCountInString(s); n > c.maxLen {
			c.maxLen = n
		}
	}
	return c
}

// Alias adds an extra text spelling that encodes to b (decode is unchanged).
func (c *Charset) Alias(text string, b byte) {
	c.toByte[text] = b
	if n := utf8.RuneCountInString(text); n > c.maxLen {
		c.maxLen = n
	}
}

// Text returns the text of one byte ("" when the byte must be escaped).
func (c *Charset) Text(b byte) string { return c.toText[b] }

// Decode converts calculator bytes to UTF-8 text, escaping unmapped bytes.
func (c *Charset) Decode(b []byte) string {
	var sb strings.Builder
	for _, x := range b {
		if s := c.toText[x]; s != "" {
			sb.WriteString(s)
		} else {
			fmt.Fprintf(&sb, "%c%02X%c", EscOpen, x, EscClose)
		}
	}
	return sb.String()
}

// EncodePrefix encodes the longest charset text (or ⟦NN⟧ escape) at the
// start of s. It returns the byte and the number of bytes of s consumed
// (0 when nothing matches).
func (c *Charset) EncodePrefix(s string) (byte, int) {
	if b, n, ok := ParseEscape(s); ok && len(b) == 1 {
		return b[0], n
	}
	ends := make([]int, 0, c.maxLen)
	for pos := 0; len(ends) < c.maxLen && pos < len(s); {
		_, n := utf8.DecodeRuneInString(s[pos:])
		pos += n
		ends = append(ends, pos)
	}
	for k := len(ends) - 1; k >= 0; k-- {
		if b, ok := c.toByte[s[:ends[k]]]; ok {
			return b, ends[k]
		}
	}
	return 0, 0
}

// Encode converts text to calculator bytes.
func (c *Charset) Encode(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))
	for len(s) > 0 {
		b, n := c.EncodePrefix(s)
		if n == 0 {
			r, _ := utf8.DecodeRuneInString(s)
			return out, fmt.Errorf("character %q has no calculator code", r)
		}
		out = append(out, b)
		s = s[n:]
	}
	return out, nil
}

// ParseEscape parses ⟦NN⟧ or ⟦NNNN⟧ at the start of s. It returns the bytes,
// the number of bytes of s consumed and ok.
func ParseEscape(s string) ([]byte, int, bool) {
	r, n := utf8.DecodeRuneInString(s)
	if r != EscOpen {
		return nil, 0, false
	}
	end := strings.IndexRune(s, EscClose)
	if end < 0 {
		return nil, 0, false
	}
	hex := s[n:end]
	if len(hex) != 2 && len(hex) != 4 {
		return nil, 0, false
	}
	var out []byte
	for i := 0; i < len(hex); i += 2 {
		var v byte
		if _, err := fmt.Sscanf(hex[i:i+2], "%02X", &v); err != nil {
			return nil, 0, false
		}
		out = append(out, v)
	}
	return out, end + utf8.RuneLen(EscClose), true
}
