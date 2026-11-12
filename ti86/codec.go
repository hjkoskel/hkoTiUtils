package ti86

import (
	"github.com/hjkoskel/hkoTiUtil/ti85family"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// Dialect is the TI-86 token engine. It implements ticalc.ProgramCodec.
var Dialect = ti85family.NewDialect(ti85family.Config{
	Model:    ticalc.ModelTI86,
	Tokens:   ti85family.StandardTokens,
	Types:    ti85family.ProgramTypes{Program: TypeProgram, Equation: TypeEquation, String: TypeString},
	TypeName: func(t ticalc.VarType) string { return Calc.TypeInfo(t).Name },
})

// Codec is Dialect as a ticalc.ProgramCodec.
var Codec ticalc.ProgramCodec = Dialect

// Convenience aliases so callers need only this package.
type (
	Token       = ti85family.Token
	ProgramKind = ti85family.ProgramKind
	ProgramInfo = ti85family.ProgramInfo
)

const (
	Tokenized = ti85family.Tokenized
	PlainText = ti85family.PlainText
	Assembly  = ti85family.Assembly
)

// Decode splits a raw TI-86 token stream into tokens.
func Decode(b []byte) ([]Token, error) { return Dialect.Decode(b) }

// Encode turns tokens back into bytes.
func Encode(toks []Token) []byte { return ti85family.Encode(toks) }

// Tokenize converts TI-BASIC text into a TI-86 token stream.
func Tokenize(src string) ([]byte, error) { return Dialect.Tokenize(src) }

// Detokenize renders a TI-86 token stream as Unicode text.
func Detokenize(b []byte, opt ticalc.DetokenizeOptions) (string, error) {
	return Dialect.Detokenize(b, opt)
}

// InspectProgram classifies a program variable.
func InspectProgram(v *ticalc.Variable) (ProgramInfo, error) { return Dialect.Inspect(v) }
