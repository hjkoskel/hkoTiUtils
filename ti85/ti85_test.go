package ti85_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/hjkoskel/hkoTiUtil/ti85"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

var update = flag.Bool("update", false, "rewrite golden files")

// TestGolden compares program text of testdata/*.85p with testdata/golden.
func TestGolden(t *testing.T) {
	files, _ := filepath.Glob("../testdata/*.85[pP]")
	if len(files) == 0 {
		t.Fatal("no TI-85 testdata")
	}
	for _, fn := range files {
		raw, err := os.ReadFile(fn)
		if err != nil {
			t.Fatal(err)
		}
		f, c, err := ticalc.Parse(raw, ticalc.ParseOptions{Strict: true})
		if err != nil {
			t.Fatalf("%s: %v", fn, err)
		}
		if c.Model() != ticalc.ModelTI85 {
			t.Fatalf("%s detected as %s", fn, c.Model())
		}
		for _, v := range f.Variables {
			got, err := c.Codec().Text(&v, ticalc.DetokenizeOptions{Strict: true})
			if err != nil {
				t.Fatalf("%s %s: %v", fn, v.Name, err)
			}
			golden := filepath.Join("../testdata/golden", v.Name+".85.txt")
			if *update {
				os.WriteFile(golden, []byte(got), 0o644)
				continue
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run go test -update)", err)
			}
			if got != string(want) {
				t.Errorf("%s: text differs from %s", v.Name, golden)
			}
		}
	}
}

// A new TI-85 file has the TI-85 header, unpadded names and parses back.
func TestNewProgramFile(t *testing.T) {
	src := "ClLCD\n0→A\nWhile A<5\nA+1→A\nOutpt(1,1,A\nEnd\nGoto X\nLbl X"
	v, err := ti85.Dialect.NewProgram("GAME", src, ticalc.TokenizeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := ti85.Marshal(&ticalc.File{Comment: "c", Variables: []ticalc.Variable{*v}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(raw, []byte("**TI85**\x1a\x0c\x00")) {
		t.Errorf("bad header % X", raw[:11])
	}
	// entry header: length 4+len("GAME") = 8, no padding
	if raw[0x37] != 8 || raw[0x37+5] != 4 {
		t.Errorf("entry header % X", raw[0x37:0x37+12])
	}
	f, err := ti85.Parse(raw, ticalc.ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ti85.Dialect.Text(&f.Variables[0], ticalc.DetokenizeOptions{Strict: true})
	if err != nil || got != src {
		t.Errorf("got %q, %v", got, err)
	}
	if ti85.Calc.TypeInfo(v.Type).Ext != "85p" {
		t.Error("wrong extension")
	}
}

// The TI-86 parser must not take TI-85 files and vice versa.
func TestDetect(t *testing.T) {
	raw, _ := os.ReadFile("../testdata/ASTRD.85P")
	if _, c, err := ticalc.Parse(raw, ticalc.ParseOptions{}); err != nil || c.Model() != ticalc.ModelTI85 {
		t.Errorf("ASTRD.85P: %v %v", c, err)
	}
}
