package ti85family

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// tokRaw is a pseudo token holding raw bytes (written as ⟦#hex⟧). It is used
// for the tail of a program whose last token is cut short.
const tokRaw = 0xFFFF

// Detokenize renders a raw token stream as Unicode text.
//
// The output is designed to be both readable and exact: Tokenize on the
// result gives back the same bytes. The only normalization is that the
// 2-byte jump cache of Goto is not kept (the calculator recomputes it).
// Anything without a readable form is written as an escape, e.g. ⟦8E33⟧ or
// ⟦FF⟧"text".
//
// In lenient mode (the default) a payload cut short at the end of the data
// is kept as a raw ⟦#hex⟧ escape and no error is returned. In strict mode
// that, and any unknown token, is an error.
func (dl *Dialect) Detokenize(b []byte, opt ticalc.DetokenizeOptions) (string, error) {
	toks, derr := dl.Decode(b)
	if derr != nil {
		if opt.Strict {
			return "", derr
		}
		last := toks[len(toks)-1]
		toks[len(toks)-1] = Token{Offset: last.Offset, Code: tokRaw, Arg: b[last.Offset:]}
	}
	pieces := make([]piece, len(toks))
	for i := range toks {
		t := &toks[i]
		if opt.Strict && t.Unknown {
			return "", fmt.Errorf("offset %d: unknown token %s", t.Offset, escape(t.Code))
		}
		pieces[i] = piece{text: dl.renderToken(t), code: t.Code}
		if d := dl.byCode[t.Code]; d != nil && t.Code != tokLbl && t.Code != tokGoto {
			pieces[i].flags = d.Flags
		}
	}
	var sb strings.Builder
	for i := range pieces {
		if i > 0 {
			sb.WriteString(dl.separator(&pieces[i-1], &pieces[i], lookahead(pieces[i:])))
		}
		sb.WriteString(pieces[i].text)
	}
	return sb.String(), nil
}

type piece struct {
	text  string
	code  uint16
	flags int
}

func (p *piece) isBreak() bool { return p.code == tokNewline || p.code == tokColon }

// separator decides what goes between two rendered tokens: a cosmetic
// space after statement words ("If X") and around infix words (" and "),
// and a required space whenever writing the two texts together would
// tokenize differently ("2" "ᴇ5" or "If" "X").
//
// ahead is the text of b and a few following tokens: the lexer may look
// further than one token ("6" "E" "-7" must not become "6E-7").
func (dl *Dialect) separator(a, b *piece, ahead string) string {
	cosmetic := (!b.isBreak() && a.flags&(FlagSpaceAfter|FlagPad) != 0) ||
		(!a.isBreak() && b.flags&FlagPad != 0)
	if cosmetic || dl.needSpace(a.text, ahead) {
		return " "
	}
	return ""
}

// lookahead joins the texts of the next few pieces.
func lookahead(ps []piece) string {
	var sb strings.Builder
	for i := 0; i < len(ps) && sb.Len() < 24; i++ {
		sb.WriteString(ps[i].text)
	}
	return sb.String()
}

// needSpace reports whether a+b would not lex a as the first token.
func (dl *Dialect) needSpace(a, b string) bool {
	l := lexer{dl: dl, s: a + b}
	_, ok, err := l.next()
	return !ok || err != nil || l.i != len(a)
}

func (dl *Dialect) renderToken(t *Token) string {
	if t.Code == tokRaw {
		return string(ticalc.EscOpen) + "#" + strings.ToUpper(hex.EncodeToString(t.Arg)) + string(ticalc.EscClose)
	}
	if t.Unknown {
		return escape(t.Code)
	}
	if kindOf(t.Code) == pkNone {
		if d := dl.byCode[t.Code]; d != nil {
			return d.Text
		}
		return escape(t.Code)
	}
	if s := dl.bareText(t); s != "" && dl.lexesTo(s, t) {
		return s
	}
	return dl.escapeForm(t)
}

// bareText is the natural text of a payload token ("" when it has none).
func (dl *Dialect) bareText(t *Token) string {
	switch {
	case t.Code == tokString:
		return dl.quote(t.Arg, t.Open)
	case t.Code == tokNumber:
		return dl.cs.Decode(t.Arg)
	case t.Code == tokLbl:
		return "Lbl " + dl.cs.Decode(t.Arg)
	case t.Code == tokGoto:
		return "Goto " + dl.cs.Decode(t.Arg)
	case kindOf(t.Code) == pkLenName || kindOf(t.Code) == pkFixName:
		return dl.cs.Decode(t.Arg)
	case t.Code == tokAsmPrgm:
		return "AsmPrgm" + dl.textCS.Decode(t.Arg)
	}
	return ""
}

// lexesTo checks that s tokenizes back to exactly t.
func (dl *Dialect) lexesTo(s string, t *Token) bool {
	l := lexer{dl: dl, s: s}
	got, ok, err := l.next()
	if !ok || err != nil || l.i != len(s) {
		return false
	}
	want := *t
	if want.Code == tokGoto {
		want.Cache = [2]byte{}
	}
	return bytes.Equal(got.AppendBytes(nil), want.AppendBytes(nil))
}

func (dl *Dialect) escapeForm(t *Token) string {
	s := escape(t.Code)
	switch kindOf(t.Code) {
	case pkRest:
		if t.Code == tokAsmCompile {
			return s + "\n" + hexLines(t.Arg, 32)
		}
		return s + dl.textCS.Decode(t.Arg)
	case pkGoto, pkExtCall:
		return s + escapeCache(t.Cache) + dl.quote(t.Arg, t.Open)
	case pkConvert:
		return s + dl.quote(t.Arg, false) + dl.quote(t.Arg2, false)
	}
	return s + dl.quote(t.Arg, t.Open)
}

func escapeCache(c [2]byte) string {
	return string(ticalc.EscOpen) + fmt.Sprintf("%02X%02X", c[0], c[1]) + string(ticalc.EscClose)
}

func escape(code uint16) string {
	if code > 0xFF {
		return string(ticalc.EscOpen) + fmt.Sprintf("%04X", code) + string(ticalc.EscClose)
	}
	return string(ticalc.EscOpen) + fmt.Sprintf("%02X", code) + string(ticalc.EscClose)
}

// quote writes "payload" with the charset; a quote character inside is
// escaped so the string ends where it should.
func (dl *Dialect) quote(b []byte, open bool) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, x := range b {
		if x == '"' {
			sb.WriteString(escape(uint16(x)))
		} else {
			sb.WriteString(dl.cs.Decode([]byte{x}))
		}
	}
	if !open {
		sb.WriteByte('"')
	}
	return sb.String()
}

func hexLines(b []byte, perLine int) string {
	var sb strings.Builder
	for i := 0; i < len(b); i += perLine {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(strings.ToUpper(hex.EncodeToString(b[i:min(len(b), i+perLine)])))
	}
	return sb.String()
}
