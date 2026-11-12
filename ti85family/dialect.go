// Package ti85family is the TI-BASIC token engine shared by the TI-85 and
// TI-86: decoding a token stream, rendering it as Unicode text, tokenizing
// text back, and the common program layout (length word, plain-text and
// edit-locked forms).
//
// The byte-level scheme (strings, numbers, names, labels, the 0x3D/0x8E
// prefixes) is the same on both calculators. What differs per model is the
// token table and the variable type codes, which a Dialect holds.
package ti85family

import (
	"fmt"
	"unicode/utf8"

	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// ProgramTypes are the variable type codes the codec needs to know.
type ProgramTypes struct {
	Program  ticalc.VarType
	Equation ticalc.VarType
	String   ticalc.VarType
}

// Config defines a dialect.
type Config struct {
	Model  ticalc.Model
	Tokens []TokenDef
	Types  ProgramTypes
	// TypeName names a type for error messages.
	TypeName func(ticalc.VarType) string
}

// Dialect is the token engine for one calculator model. It implements
// ticalc.ProgramCodec.
type Dialect struct {
	model    ticalc.Model
	types    ProgramTypes
	typeName func(ticalc.VarType) string
	byCode   map[uint16]*TokenDef
	byText   map[string]*TokenDef // every accepted spelling (Text and Alts)
	maxRunes int
	cs       *ticalc.Charset
	textCS   *ticalc.Charset
}

var _ ticalc.ProgramCodec = (*Dialect)(nil)

// NewDialect builds a dialect. It panics on duplicate codes or spellings
// (a programming error in the table).
func NewDialect(c Config) *Dialect {
	dl := &Dialect{
		model: c.Model, types: c.Types, typeName: c.TypeName,
		byCode: map[uint16]*TokenDef{}, byText: map[string]*TokenDef{},
		cs: Charset, textCS: TextCharset,
	}
	if dl.typeName == nil {
		dl.typeName = func(t ticalc.VarType) string { return fmt.Sprintf("type 0x%02X", byte(t)) }
	}
	for i := range c.Tokens {
		d := &c.Tokens[i]
		if _, dup := dl.byCode[d.Code]; dup {
			panic(fmt.Sprintf("ti85family: %s: duplicate token code %04X", c.Model, d.Code))
		}
		dl.byCode[d.Code] = d
		for _, s := range append([]string{d.Text}, d.Alts...) {
			if s == "\n" {
				continue
			}
			if _, dup := dl.byText[s]; dup {
				panic(fmt.Sprintf("ti85family: %s: duplicate token text %q", c.Model, s))
			}
			dl.byText[s] = d
			if n := utf8.RuneCountInString(s); n > dl.maxRunes {
				dl.maxRunes = n
			}
		}
	}
	return dl
}

// Model is the calculator model of the dialect.
func (dl *Dialect) Model() ticalc.Model { return dl.model }

// Token returns the table entry of a code (nil when unknown).
func (dl *Dialect) Token(code uint16) *TokenDef { return dl.byCode[code] }
