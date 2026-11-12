package ticalc_test

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

var testContainer = &ticalc.Container{
	Model:     "TEST",
	Signature: "**TI86**",
	PadNames:  true,
	Charset:   ticalc.NewCharset(true, map[byte]string{0xC1: "θ"}),
}

func TestContainerRoundTrip(t *testing.T) {
	in := &ticalc.File{
		Comment: "hello",
		Variables: []ticalc.Variable{
			{Type: 0x12, Name: "PROG", Data: []byte{2, 0, 0xE9, 0x33}},
			{Type: 0x00, Name: "θ", Data: bytes.Repeat([]byte{1}, 10)},
		},
	}
	for _, pad := range []bool{true, false} {
		c := *testContainer
		c.PadNames = pad
		raw, err := c.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		out, err := c.Parse(raw, ticalc.ParseOptions{Strict: true})
		if err != nil {
			t.Fatalf("pad=%v: %v", pad, err)
		}
		if out.Comment != "hello" || len(out.Variables) != 2 {
			t.Fatalf("pad=%v: got %+v", pad, out)
		}
		for i, v := range out.Variables {
			if v.Name != in.Variables[i].Name || v.Type != in.Variables[i].Type || !bytes.Equal(v.Data, in.Variables[i].Data) {
				t.Errorf("pad=%v var %d: got %+v", pad, i, v)
			}
		}
	}
}

func TestContainerLenient(t *testing.T) {
	raw, _ := testContainer.Marshal(&ticalc.File{Variables: []ticalc.Variable{{Type: 0, Name: "A", Data: []byte{1, 2}}}})
	raw[len(raw)-1] ^= 0xFF // break checksum
	if _, err := testContainer.Parse(raw, ticalc.ParseOptions{Strict: true}); err == nil {
		t.Error("strict mode accepted bad checksum")
	}
	f, err := testContainer.Parse(raw, ticalc.ParseOptions{})
	if err != nil || len(f.Warnings) != 1 || !strings.Contains(f.Warnings[0].Error(), "checksum") {
		t.Errorf("lenient: err=%v warnings=%v", err, f.Warnings)
	}
	// Truncated input never panics.
	for n := 0; n < len(raw); n++ {
		testContainer.Parse(raw[:n], ticalc.ParseOptions{})
	}
}

func TestCharsetEscapes(t *testing.T) {
	cs := testContainer.Charset
	in := []byte{'A', 0xC1, 0x00, 0xFE}
	s := cs.Decode(in)
	if s != "Aθ⟦00⟧⟦FE⟧" {
		t.Errorf("decode %q", s)
	}
	out, err := cs.Encode(s)
	if err != nil || !bytes.Equal(out, in) {
		t.Errorf("encode % X %v", out, err)
	}
}

func TestDetectUnknown(t *testing.T) {
	_, err := ticalc.Detect([]byte("**TI99**\x1a\x0a\x00"))
	if err == nil || !strings.Contains(err.Error(), "TI99") {
		t.Errorf("got %v", err)
	}
}

func FuzzContainerParse(f *testing.F) {
	raw, _ := testContainer.Marshal(&ticalc.File{Variables: []ticalc.Variable{{Type: 0x12, Name: "P", Data: []byte{1, 0, 0x33}}}})
	f.Add(raw)
	f.Fuzz(func(t *testing.T, b []byte) {
		testContainer.Parse(b, ticalc.ParseOptions{})
	})
}

func TestImageText(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 3, 3))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
	img.Pix[0] = 0     // (0,0)
	img.Pix[1*3+1] = 0 // (1,1)
	img.Pix[0*3+2] = 0 // (2,0)
	img.Pix[1*3+2] = 0 // (2,1)
	img.Pix[2*3+0] = 0 // (0,2)
	want := "▀▄█\n▀\n"
	if got := ticalc.ImageText(img); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
