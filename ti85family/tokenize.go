package ti85family

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// Tokenize converts TI-BASIC source text into a raw token stream.
//
// It accepts the Unicode text produced by Detokenize and also ASCII
// spellings of symbols: "->" (→), "!=" or "=/=" (≠), "<=" (≤), ">=" (≥),
// ">Dec" (▶Dec), "/_" (∠), "~" (negation) and so on. Whitespace between
// tokens is insignificant (there is no space token), except that it
// separates words: "If X" is If followed by X, "IfX" is a variable.
// Escapes ⟦NN⟧ / ⟦NNNN⟧ insert raw bytes/tokens.
func (dl *Dialect) Tokenize(src string) ([]byte, error) {
	toks, err := dl.Lex(src)
	if err != nil {
		return nil, err
	}
	return Encode(toks), nil
}

// Lex converts source text into tokens.
func (dl *Dialect) Lex(src string) ([]Token, error) {
	l := lexer{dl: dl, s: src}
	var out []Token
	for {
		t, ok, err := l.next()
		if err != nil {
			line := 1 + strings.Count(src[:l.i], "\n")
			return out, fmt.Errorf("line %d: %w", line, err)
		}
		if !ok {
			return out, nil
		}
		out = append(out, t)
	}
}

type lexer struct {
	dl *Dialect
	s  string
	i  int
}

func (l *lexer) peek() rune {
	r, _ := utf8.DecodeRuneInString(l.s[l.i:])
	return r
}

func (l *lexer) skipSpace() {
	for l.i < len(l.s) {
		switch l.s[l.i] {
		case ' ', '\t', '\r':
			l.i++
		default:
			return
		}
	}
}

// next returns the next token; ok is false at end of input.
func (l *lexer) next() (Token, bool, error) {
	l.skipSpace()
	if l.i >= len(l.s) {
		return Token{}, false, nil
	}
	start := l.i
	t := Token{Offset: start}
	rest := l.s[l.i:]
	r := l.peek()

	switch {
	case r == '\n':
		l.i++
		t.Code = tokNewline
		return t, true, nil

	case strings.HasPrefix(rest, string(ticalc.EscOpen)+"#"):
		end := strings.IndexRune(rest, ticalc.EscClose)
		if end < 0 {
			return t, false, fmt.Errorf("unterminated raw escape")
		}
		b, err := hex.DecodeString(rest[len(string(ticalc.EscOpen))+1 : end])
		if err != nil {
			return t, false, fmt.Errorf("raw escape: %w", err)
		}
		l.i += end + len(string(ticalc.EscClose))
		t.Code = tokRaw
		t.Arg = b
		return t, true, nil

	case r == ticalc.EscOpen:
		b, n, ok := ticalc.ParseEscape(rest)
		if !ok {
			return t, false, fmt.Errorf("bad escape %q", clip(rest))
		}
		l.i += n
		if len(b) == 2 {
			t.Code = uint16(b[0])<<8 | uint16(b[1])
		} else {
			t.Code = uint16(b[0])
		}
		return l.escapedPayload(t)

	case r == '"':
		l.i++
		t.Code = tokString
		var err error
		t.Arg, t.Open, err = l.quoted(l.dl.cs)
		return t, err == nil, err
	}

	if d, n := l.dl.matchToken(rest); d != nil {
		l.i += n
		t.Code = d.Code
		switch d.Code {
		case tokLbl, tokGoto:
			l.skipSpace()
			j := l.i
			for j < len(l.s) && !strings.ContainsRune(" \t\r\n:", rune(l.s[j])) {
				j++
			}
			arg, err := l.dl.cs.Encode(l.s[l.i:j])
			if err != nil {
				return t, false, fmt.Errorf("label: %w", err)
			}
			t.Arg = arg
			l.i = j
		case tokAsmPrgm:
			arg, err := l.dl.textCS.Encode(l.s[l.i:])
			if err != nil {
				return t, false, fmt.Errorf("AsmPrgm: %w", err)
			}
			t.Arg = arg
			l.i = len(l.s)
		}
		return t, true, nil
	}

	if n := matchNumber(rest); n > 0 {
		arg, err := l.dl.encodeNumber(rest[:n])
		if err != nil {
			return t, false, err
		}
		l.i += n
		t.Code = tokNumber
		t.Arg = arg
		return t, true, nil
	}

	if n := matchIdent(rest); n > 0 {
		word := rest[:n]
		l.i += n
		if d, ok := l.dl.byText[word]; ok {
			t.Code = d.Code
			return t, true, nil
		}
		arg, err := l.dl.cs.Encode(word)
		if err != nil {
			return t, false, fmt.Errorf("name %q: %w", word, err)
		}
		t.Code = nameCode(word, len(arg))
		if t.Code == 0 {
			return t, false, fmt.Errorf("name %q is longer than 8 characters", word)
		}
		t.Arg = arg
		return t, true, nil
	}

	return t, false, fmt.Errorf("unexpected character %q", r)
}

// escapedPayload reads the payload that follows an escaped payload token
// (⟦2D⟧"…", ⟦E1⟧⟦CCCC⟧"…", ⟦3E⟧"from""to", ⟦8E28⟧ hex …).
func (l *lexer) escapedPayload(t Token) (Token, bool, error) {
	k := kindOf(t.Code)
	if k == pkNone {
		return t, true, nil
	}
	if k == pkRest {
		rest := l.s[l.i:]
		l.i = len(l.s)
		if t.Code == tokAsmCompile {
			b, err := hex.DecodeString(strings.Join(strings.Fields(rest), ""))
			if err != nil {
				return t, false, fmt.Errorf("compiled asm hex: %w", err)
			}
			t.Arg = b
			return t, true, nil
		}
		b, err := l.dl.textCS.Encode(rest)
		t.Arg = b
		return t, err == nil, err
	}
	if k == pkGoto || k == pkExtCall {
		b, n, ok := ticalc.ParseEscape(l.s[l.i:])
		if !ok || len(b) != 2 {
			return t, false, fmt.Errorf("expected ⟦CCCC⟧ after token %04X", t.Code)
		}
		copy(t.Cache[:], b)
		l.i += n
	}
	readQuoted := func() ([]byte, bool, error) {
		if l.i >= len(l.s) || l.s[l.i] != '"' {
			return nil, false, fmt.Errorf("expected quoted payload after token %04X", t.Code)
		}
		l.i++
		return l.quoted(l.dl.cs)
	}
	var err error
	if t.Arg, t.Open, err = readQuoted(); err != nil {
		return t, false, err
	}
	if k == pkConvert {
		if t.Arg2, _, err = readQuoted(); err != nil {
			return t, false, err
		}
	}
	if k == pkFixName && len(t.Arg) != int(t.Code-tokName1)+1 {
		return t, false, fmt.Errorf("name token %02X needs %d bytes", t.Code, t.Code-tokName1+1)
	}
	return t, true, nil
}

// quoted reads charset text up to a closing quote (consumed), newline (not
// consumed) or end of input (open=true).
func (l *lexer) quoted(cs *ticalc.Charset) ([]byte, bool, error) {
	var out []byte
	for l.i < len(l.s) {
		switch l.s[l.i] {
		case '"':
			l.i++
			return out, false, nil
		case '\n':
			return out, false, nil
		}
		b, n := cs.EncodePrefix(l.s[l.i:])
		if n == 0 {
			r, _ := utf8.DecodeRuneInString(l.s[l.i:])
			return out, false, fmt.Errorf("character %q has no calculator code", r)
		}
		out = append(out, b)
		l.i += n
	}
	return out, true, nil
}

// matchToken finds the longest token spelling at the start of s. A spelling
// that ends in a word character only matches when the next character does
// not continue the word ("sin X" matches sin, "sinX" is a variable).
func (dl *Dialect) matchToken(s string) (*TokenDef, int) {
	ends := make([]int, 0, dl.maxRunes)
	for pos := 0; len(ends) < dl.maxRunes && pos < len(s); {
		_, n := utf8.DecodeRuneInString(s[pos:])
		pos += n
		ends = append(ends, pos)
	}
	for k := len(ends) - 1; k >= 0; k-- {
		e := ends[k]
		d, ok := dl.byText[s[:e]]
		if !ok {
			continue
		}
		last, _ := utf8.DecodeLastRuneInString(s[:e])
		next, _ := utf8.DecodeRuneInString(s[e:])
		if isWordRune(last) && e < len(s) && isWordRune(next) {
			continue
		}
		return d, e
	}
	return nil, 0
}

// isWordRune reports runes that form names and words.
func isWordRune(r rune) bool {
	switch {
	case r >= '0' && r <= '9', r == '\'':
		return true
	case r >= '₀' && r <= '₉':
		return true
	case unicode.Is(unicode.Lm, r):
		return false // ʳ ᵀ ˣ ᵇ … are operators/suffixes
	case unicode.IsLetter(r), unicode.Is(unicode.Mn, r):
		return true
	}
	return false
}

func matchIdent(s string) int {
	n := 0
	for i, r := range s {
		if i == 0 && (r == '\'' || (r >= '0' && r <= '9') || (r >= '₀' && r <= '₉')) {
			return 0
		}
		if !isWordRune(r) {
			break
		}
		n = i + utf8.RuneLen(r)
	}
	// Σx² and Σy² are statistics variables that end in ².
	if n > 0 && strings.HasPrefix(s[n:], "²") {
		if _, ok := statNames[s[:n]+"²"]; ok {
			n += len("²")
		}
	}
	return n
}

// Number: digits, '.', hex digits Ａ-Ｆ, exponent ᴇ (or E before a digit),
// negation ⁻ (or -) right after the exponent, and a base suffix ᵇ ᵒ ᵈ ʰ.
func matchNumber(s string) int {
	i := 0
	digits := false
	for i < len(s) {
		r, n := utf8.DecodeRuneInString(s[i:])
		switch {
		case r >= '0' && r <= '9', r == '.', r >= 'Ａ' && r <= 'Ｆ':
			digits = true
			i += n
		case r == 'ᴇ' || (r == 'E' && digits && expFollows(s[i+n:])):
			i += n
			if strings.HasPrefix(s[i:], "⁻") {
				i += len("⁻")
			} else if strings.HasPrefix(s[i:], "-") {
				i++
			}
			digits = true
		case r == 'ᵇ' || r == 'ᵒ' || r == 'ᵈ' || r == 'ʰ':
			if !digits {
				return 0
			}
			return i + n
		default:
			if !digits {
				return 0
			}
			return i
		}
	}
	if !digits {
		return 0
	}
	return i
}

func expFollows(s string) bool {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "⁻"), "-")
	return len(s) > 0 && s[0] >= '0' && s[0] <= '9'
}

func (dl *Dialect) encodeNumber(s string) ([]byte, error) {
	var out []byte
	for len(s) > 0 {
		switch {
		case s[0] == 'E':
			out = append(out, 0x1B)
			s = s[1:]
		case s[0] == '-':
			out = append(out, 0x1A)
			s = s[1:]
		default:
			b, n := dl.cs.EncodePrefix(s)
			if n == 0 {
				return nil, fmt.Errorf("bad number %q", s)
			}
			out = append(out, b)
			s = s[n:]
		}
	}
	return out, nil
}

// Names stored with the 0x32 token.
var qNames = regexp.MustCompile(`^(x|y|t|r|θ|QI?[1-9]|exp|eqn)$`)

// Names stored with the 0x3B token (graph equations).
var yNames = regexp.MustCompile(`^(y[1-9][0-9]?|xt[1-9][0-9]?|yt[1-9][0-9]?|r[1-9][0-9]?|Q'[1-9])$`)

// Names stored with the 0x3C token (system/statistics results).
var statNames = map[string]bool{
	"Ans": true, "a": true, "b": true, "corr": true, "maxX": true, "maxY": true,
	"Med": true, "minX": true, "minY": true, "n": true, "PRegC": true,
	"Qrtl1": true, "Qrtl3": true, "RegEq": true, "Sx": true, "Sy": true,
	"σx": true, "σy": true, "Σx": true, "Σx²": true, "Σxy": true, "Σy": true,
	"Σy²": true, "tolMet": true, "x̄": true, "ȳ": true,
}

// nameCode picks the token used to store a variable name. It returns 0 when
// the name is too long.
func nameCode(name string, nbytes int) uint16 {
	switch {
	case qNames.MatchString(name):
		return tokNameQ
	case yNames.MatchString(name):
		return tokNameY
	case statNames[name]:
		return tokNameStat
	case nbytes >= 1 && nbytes <= 8:
		return uint16(tokName1 + nbytes - 1)
	}
	return 0
}

func clip(s string) string {
	if len(s) > 12 {
		return s[:12] + "…"
	}
	return s
}
