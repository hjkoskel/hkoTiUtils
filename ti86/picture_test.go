package ti86_test

import (
	"os"
	"testing"

	"github.com/hjkoskel/hkoTiUtil/ti86"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// TestPicture decodes testdata/TT3DPIC.86i and compares its block drawing
// with testdata/golden/TT3DPic.pic.txt.
func TestPicture(t *testing.T) {
	raw, err := os.ReadFile("../testdata/TT3DPIC.86i")
	if err != nil {
		t.Fatal(err)
	}
	f, err := ti86.Parse(raw, ticalc.ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	img, err := ti86.Calc.Image(&f.Variables[0])
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 128 || b.Dy() != 63 {
		t.Errorf("bounds %v", b)
	}
	golden := "../testdata/golden/TT3DPic.pic.txt"
	got := ticalc.ImageText(img)
	if *update {
		os.WriteFile(golden, []byte(got), 0o644)
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Error("picture differs from golden")
	}
	if _, err := ti86.Calc.Image(&ticalc.Variable{Type: ti86.TypeProgram}); err != ticalc.ErrNotImage {
		t.Errorf("program as image: %v", err)
	}
}
