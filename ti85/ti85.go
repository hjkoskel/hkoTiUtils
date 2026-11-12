// Package ti85 implements ticalc.Calculator for the TI-85: reading and
// writing *.85? variable files and converting TI-BASIC programs to Unicode
// text and back.
//
// The TI-85 uses the same file container, font and token encoding as the
// TI-86 (the TI-86 extended it), so the token engine comes from
// ti85family. Differences: the "**TI85**" signature with 1A 0C 00, names
// stored without padding, and .85x extensions.
//
// Importing the package registers the TI-85 in the ticalc registry:
//
//	import _ "github.com/hjkoskel/hkoTiUtil/ti85"
package ti85

import (
	"fmt"
	"image"

	"github.com/hjkoskel/hkoTiUtil/ti85family"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// TI-85 variable type codes.
const (
	TypeReal            ticalc.VarType = 0x00
	TypeComplex         ticalc.VarType = 0x01
	TypeRealVector      ticalc.VarType = 0x02
	TypeComplexVector   ticalc.VarType = 0x03
	TypeRealList        ticalc.VarType = 0x04
	TypeComplexList     ticalc.VarType = 0x05
	TypeRealMatrix      ticalc.VarType = 0x06
	TypeComplexMatrix   ticalc.VarType = 0x07
	TypeRealConstant    ticalc.VarType = 0x08
	TypeComplexConstant ticalc.VarType = 0x09
	TypeEquation        ticalc.VarType = 0x0A
	TypeString          ticalc.VarType = 0x0C
	TypeFunctionGDB     ticalc.VarType = 0x0D
	TypePolarGDB        ticalc.VarType = 0x0E
	TypeParametricGDB   ticalc.VarType = 0x0F
	TypeDiffEqGDB       ticalc.VarType = 0x10
	TypePicture         ticalc.VarType = 0x11
	TypeProgram         ticalc.VarType = 0x12
	TypeDirectory       ticalc.VarType = 0x15
	TypeFunctionRange   ticalc.VarType = 0x17
	TypePolarRange      ticalc.VarType = 0x18
	TypeParametricRange ticalc.VarType = 0x19
	TypeDiffEqRange     ticalc.VarType = 0x1A
	TypeSavedRange      ticalc.VarType = 0x1B
	TypeBackup          ticalc.VarType = 0x1D
	TypeUnknown         ticalc.VarType = 0x1E
)

var typeInfos = map[ticalc.VarType]ticalc.TypeInfo{
	TypeReal:            {Name: "Real", Ext: "85n"},
	TypeComplex:         {Name: "Complex", Ext: "85c"},
	TypeRealVector:      {Name: "Real Vector", Ext: "85v"},
	TypeComplexVector:   {Name: "Complex Vector", Ext: "85v"},
	TypeRealList:        {Name: "Real List", Ext: "85l"},
	TypeComplexList:     {Name: "Complex List", Ext: "85l"},
	TypeRealMatrix:      {Name: "Real Matrix", Ext: "85m"},
	TypeComplexMatrix:   {Name: "Complex Matrix", Ext: "85m"},
	TypeRealConstant:    {Name: "Real Constant", Ext: "85k"},
	TypeComplexConstant: {Name: "Complex Constant", Ext: "85k"},
	TypeEquation:        {Name: "Equation", Ext: "85e", Text: true},
	TypeString:          {Name: "String", Ext: "85s", Text: true},
	TypeFunctionGDB:     {Name: "Function GDB", Ext: "85d"},
	TypePolarGDB:        {Name: "Polar GDB", Ext: "85d"},
	TypeParametricGDB:   {Name: "Parametric GDB", Ext: "85d"},
	TypeDiffEqGDB:       {Name: "Differential Equation GDB", Ext: "85d"},
	TypePicture:         {Name: "Picture", Ext: "85i"},
	TypeProgram:         {Name: "Program", Ext: "85p", Text: true, Program: true},
	TypeDirectory:       {Name: "Directory"},
	TypeFunctionRange:   {Name: "Function Range", Ext: "85r"},
	TypePolarRange:      {Name: "Polar Range", Ext: "85r"},
	TypeParametricRange: {Name: "Parametric Range", Ext: "85r"},
	TypeDiffEqRange:     {Name: "Differential Equation Range", Ext: "85r"},
	TypeSavedRange:      {Name: "Saved Range (ZRCL)", Ext: "85r"},
	TypeBackup:          {Name: "Memory Backup", Ext: "85b"},
	TypeUnknown:         {Name: "Unknown"},
}

// Tokens is the TI-85 token table. TI-85 programs in the example corpus
// decode and round-trip with the shared TI-85/86 table; they only use
// one-byte and 3D xx tokens. The 8E xx commands were added by the TI-86
// and are kept so that such files still decode.
// TODO: check against a TI-85 token reference and drop TI-86-only tokens.
var Tokens = ti85family.StandardTokens

// Charset is the TI-85 font (the same as the TI-86 font).
var Charset = ti85family.Charset

// Dialect is the TI-85 token engine. It implements ticalc.ProgramCodec.
var Dialect = ti85family.NewDialect(ti85family.Config{
	Model:    ticalc.ModelTI85,
	Tokens:   Tokens,
	Types:    ti85family.ProgramTypes{Program: TypeProgram, Equation: TypeEquation, String: TypeString},
	TypeName: func(t ticalc.VarType) string { return Calc.TypeInfo(t).Name },
})

// Calc is the TI-85 implementation of ticalc.Calculator.
var Calc = &calculator{
	container: &ticalc.Container{
		Model:     ticalc.ModelTI85,
		Signature: "**TI85**",
		Magic:     [3]byte{0x1A, 0x0C, 0x00},
		PadNames:  false,
		Charset:   Charset,
	},
}

func init() { ticalc.Register(Calc) }

type calculator struct {
	container *ticalc.Container
}

var (
	_ ticalc.Calculator = (*calculator)(nil)
	_ ticalc.Describer  = (*calculator)(nil)
)

func (c *calculator) Model() ticalc.Model                       { return ticalc.ModelTI85 }
func (c *calculator) Detect(raw []byte) bool                    { return c.container.Detect(raw) }
func (c *calculator) Codec() ticalc.ProgramCodec                { return Dialect }
func (c *calculator) Details(v *ticalc.Variable) ticalc.Details { return Dialect.Details(v) }
func (c *calculator) Marshal(f *ticalc.File) ([]byte, error)    { return c.container.Marshal(f) }
func (c *calculator) Parse(raw []byte, opt ticalc.ParseOptions) (*ticalc.File, error) {
	return c.container.Parse(raw, opt)
}

func (c *calculator) TypeInfo(t ticalc.VarType) ticalc.TypeInfo {
	if i, ok := typeInfos[t]; ok {
		return i
	}
	return ticalc.TypeInfo{Name: fmt.Sprintf("Type 0x%02X", byte(t)), Ext: "85?"}
}

// Parse parses a TI-85 file.
func Parse(raw []byte, opt ticalc.ParseOptions) (*ticalc.File, error) { return Calc.Parse(raw, opt) }

// Marshal writes a TI-85 file.
func Marshal(f *ticalc.File) ([]byte, error) { return Calc.Marshal(f) }

var _ ticalc.Imager = (*calculator)(nil)

// Image decodes a picture variable (see ti85family.Picture).
func (c *calculator) Image(v *ticalc.Variable) (image.Image, error) {
	if v.Type != TypePicture {
		return nil, ticalc.ErrNotImage
	}
	return ti85family.Picture(v.Data)
}
