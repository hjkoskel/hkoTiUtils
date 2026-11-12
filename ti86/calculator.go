package ti86

import (
	"fmt"
	"image"

	"github.com/hjkoskel/hkoTiUtil/ti85family"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// Calc is the TI-86 implementation of ticalc.Calculator.
var Calc = &calculator{
	container: &ticalc.Container{
		Model:     ticalc.ModelTI86,
		Signature: "**TI86**",
		PadNames:  true,
		Charset:   Charset,
	},
}

func init() { ticalc.Register(Calc) }

type calculator struct {
	container *ticalc.Container
}

var _ ticalc.Calculator = (*calculator)(nil)

func (c *calculator) Model() ticalc.Model        { return ticalc.ModelTI86 }
func (c *calculator) Detect(raw []byte) bool     { return c.container.Detect(raw) }
func (c *calculator) Codec() ticalc.ProgramCodec { return Codec }

func (c *calculator) Parse(raw []byte, opt ticalc.ParseOptions) (*ticalc.File, error) {
	return c.container.Parse(raw, opt)
}

func (c *calculator) Marshal(f *ticalc.File) ([]byte, error) {
	return c.container.Marshal(f)
}

func (c *calculator) TypeInfo(t ticalc.VarType) ticalc.TypeInfo {
	if i, ok := typeInfos[t]; ok {
		return i
	}
	return ticalc.TypeInfo{Name: fmt.Sprintf("Type 0x%02X", byte(t)), Ext: "86?"}
}

// Parse parses a TI-86 file.
func Parse(raw []byte, opt ticalc.ParseOptions) (*ticalc.File, error) { return Calc.Parse(raw, opt) }

// Marshal writes a TI-86 file.
func Marshal(f *ticalc.File) ([]byte, error) { return Calc.Marshal(f) }

var _ ticalc.Describer = (*calculator)(nil)

// Details reports program kind and edit lock.
func (c *calculator) Details(v *ticalc.Variable) ticalc.Details { return Dialect.Details(v) }

var _ ticalc.Imager = (*calculator)(nil)

// Image decodes a picture variable (see ti85family.Picture).
func (c *calculator) Image(v *ticalc.Variable) (image.Image, error) {
	if v.Type != TypePicture {
		return nil, ticalc.ErrNotImage
	}
	return ti85family.Picture(v.Data)
}
