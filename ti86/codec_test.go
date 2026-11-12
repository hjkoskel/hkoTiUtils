package ti86_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hjkoskel/hkoTiUtil/ti86"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

var update = flag.Bool("update", false, "rewrite golden files")

// TestGolden compares program text of testdata/*.86p with testdata/golden.
func TestGolden(t *testing.T) {
	files, _ := filepath.Glob("../testdata/*.86[pP]")
	if len(files) == 0 {
		t.Fatal("no testdata")
	}
	for _, fn := range files {
		raw, err := os.ReadFile(fn)
		if err != nil {
			t.Fatal(err)
		}
		f, err := ti86.Parse(raw, ticalc.ParseOptions{Strict: true})
		if err != nil {
			t.Fatalf("%s: %v", fn, err)
		}
		for _, v := range f.Variables {
			got, err := ti86.Codec.Text(&v, ticalc.DetokenizeOptions{Strict: true})
			if err != nil {
				t.Fatalf("%s %s: %v", fn, v.Name, err)
			}
			golden := filepath.Join("../testdata/golden", v.Name+".txt")
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

func TestTokenize(t *testing.T) {
	cases := []struct {
		src  string
		want []byte
	}{
		{"Goto LOPPU", append([]byte{0xE1, 0, 0}, "LOPPU\x00"...)},
		{"Lbl A1", append([]byte{0xE0}, "A1\x00"...)},
		{`Disp "HI"`, []byte{0xE9, 0x2D, 'H', 'I', 0}},
		{`"open`, []byte{0x2D, 'o', 'p', 'e', 'n'}},
		{"1.5ᴇ⁻3", []byte{0x44, '1', '.', '5', 0x1B, 0x1A, '3', 0}},
		{"1.5E-3", []byte{0x44, '1', '.', '5', 0x1B, 0x1A, '3', 0}},
		{"ＦＦʰ", []byte{0x44, 0x0F, 0x0F, 0x04, 0}},
		{"x", []byte{0x32, 1, 'x'}},
		{"y1", []byte{0x3B, 2, 'y', '1'}},
		{"Ans", []byte{0x3C, 3, 'A', 'n', 's'}},
		{"ABC", []byte{0x35, 'A', 'B', 'C'}},
		{"θ", []byte{0x32, 1, 0xC1}},
		{"sin X", []byte{0xAA, 0x33, 'X'}},
		{"sinX", []byte{0x36, 's', 'i', 'n', 'X'}},
		{"Text(1,1", []byte{0x8E, 0x13, 0x10, 0x44, '1', 0, 0x2F, 0x44, '1', 0}},
		{"2X", []byte{0x44, '2', 0, 0x33, 'X'}},
		{"xMin", []byte{0x3D, 0x04}},
		{"A and B", []byte{0x33, 'A', 0x40, 0x33, 'B'}},
		{"₁₀^(2)", []byte{0xA9, 0x10, 0x44, '2', 0, 0x11}},
		{"⟦8E33⟧", []byte{0x8E, 0x33}},
		{`⟦FF⟧"note"`, []byte{0xFF, 'n', 'o', 't', 'e', 0}},
		{"⟦#0102⟧", []byte{1, 2}},
		{"A\nB:C", []byte{0x33, 'A', 0x6F, 0x33, 'B', 0x6E, 0x33, 'C'}},
	}
	for _, c := range cases {
		got, err := ti86.Tokenize(c.src)
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
			continue
		}
		if !bytes.Equal(got, c.want) {
			t.Errorf("%q:\n got  % X\n want % X", c.src, got, c.want)
		}
	}
}

// ASCII spellings must give the same tokens as the Unicode ones.
func TestTokenizeASCII(t *testing.T) {
	pairs := [][2]string{
		{"5->X", "5→X"},
		{"If A!=B", "If A≠B"},
		{"If A=/=B", "If A≠B"},
		{"A<=B", "A≤B"},
		{"A>=B", "A≥B"},
		{"X>Dec", "X▶Dec"},
		{"5/_30", "5∠30"},
		{"~5", "⁻5"},
		{"Eq>St(y1,S)", "Eq▶St(y1,S)"},
		{"li>vc L", "li▶vc L"},
	}
	for _, p := range pairs {
		a, err := ti86.Tokenize(p[0])
		if err != nil {
			t.Fatalf("%q: %v", p[0], err)
		}
		u, err := ti86.Tokenize(p[1])
		if err != nil {
			t.Fatalf("%q: %v", p[1], err)
		}
		if !bytes.Equal(a, u) {
			t.Errorf("%q and %q differ: % X vs % X", p[0], p[1], a, u)
		}
	}
}

func TestTokenizeErrors(t *testing.T) {
	for _, src := range []string{"ABCDEFGHI", "⟦zz⟧", "A#B"} {
		if _, err := ti86.Tokenize(src); err == nil {
			t.Errorf("%q: expected error", src)
		}
	}
}

// A program written as text survives NewProgram → file → Parse → Text.
func TestNewProgramFile(t *testing.T) {
	src := "ClLCD\nFor(I,1,10\nOutpt(1,1,\"Hello→\"\nIf I≠5 and I>2\nDisp I²\nEnd\nGoto A\nLbl A"
	v, err := ti86.Codec.NewProgram("HELLO", src, ticalc.TokenizeOptions{EditLocked: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := ti86.Marshal(&ticalc.File{Comment: "test", Variables: []ticalc.Variable{*v}})
	if err != nil {
		t.Fatal(err)
	}
	f, err := ti86.Parse(raw, ticalc.ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if f.Comment != "test" || len(f.Variables) != 1 || f.Variables[0].Name != "HELLO" {
		t.Fatalf("unexpected file %+v", f)
	}
	pi, _ := ti86.InspectProgram(&f.Variables[0])
	if !pi.EditLocked || pi.Kind != ti86.Tokenized {
		t.Errorf("program info %+v", pi)
	}
	got, err := ti86.Codec.Text(&f.Variables[0], ticalc.DetokenizeOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Errorf("text changed:\n%s\n---\n%s", got, src)
	}
}

func TestPlainTextProgram(t *testing.T) {
	body := append([]byte{0, 0}, "Disp \"A\"\xd65\x1cX"...)
	data := append([]byte{byte(len(body)), 0}, body...)
	v := ticalc.Variable{Type: ti86.TypeProgram, Name: "P", Data: data}
	pi, _ := ti86.InspectProgram(&v)
	if pi.Kind != ti86.PlainText || !pi.EditLocked {
		t.Errorf("info %+v", pi)
	}
	got, _ := ti86.Codec.Text(&v, ticalc.DetokenizeOptions{})
	if got != "Disp \"A\"\n5→X" {
		t.Errorf("got %q", got)
	}
}

func TestDetokenizeSpacing(t *testing.T) {
	cases := map[string]string{
		"If X":     "If X",
		"6 E-7":    "6 E-7", // must not turn into the number 6ᴇ⁻7
		"2X":       "2X",
		"A or B":   "A or B",
		"(A)or(B)": "(A) or (B)",
		"ℯ ^2":     "ℯ ^2", // constant ℯ then ^, not the ℯ^ function
	}
	for src, want := range cases {
		b, err := ti86.Tokenize(src)
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		got, _ := ti86.Detokenize(b, ticalc.DetokenizeOptions{})
		if got != want {
			t.Errorf("%q: got %q want %q", src, got, want)
		}
		if strings.Contains(src, "E-7") {
			if b2, _ := ti86.Tokenize(got); !bytes.Equal(b, b2) {
				t.Errorf("%q does not round trip", src)
			}
		}
	}
}

// Every single byte and every pair of bytes must survive Decode → Encode.
func TestCharsetPairs(t *testing.T) {
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			in := []byte{byte(a), byte(b)}
			out, err := ti86.Charset.Encode(ti86.Charset.Decode(in))
			if err != nil || !bytes.Equal(in, out) {
				t.Fatalf("% X -> %q -> % X (%v)", in, ti86.Charset.Decode(in), out, err)
			}
		}
	}
}
