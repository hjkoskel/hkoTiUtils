// Corpus tests run over the local example collection in download/ (not in
// git) for every registered calculator model. They are skipped when the
// directory is missing.
package hkotiutil_test

import (
	"bytes"
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/hjkoskel/hkoTiUtil/ti85family"
	"github.com/hjkoskel/hkoTiUtil/tiarchive"
	"github.com/hjkoskel/hkoTiUtil/ticalc"

	_ "github.com/hjkoskel/hkoTiUtil/ti85"
	_ "github.com/hjkoskel/hkoTiUtil/ti86"
)

const corpusDir = "download"

type item struct {
	e    tiarchive.Entry
	f    *ticalc.File
	calc ticalc.Calculator
}

func walkCorpus(t *testing.T, fn func(item)) {
	t.Helper()
	if _, err := os.Stat(corpusDir); err != nil {
		t.Skip("no corpus directory")
	}
	err := tiarchive.Walk(corpusDir, tiarchive.Options{}, func(e tiarchive.Entry) error {
		f, c, err := ticalc.Parse(e.Data, ticalc.ParseOptions{})
		if err != nil {
			return nil // unsupported models and broken files
		}
		fn(item{e, f, c})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// normalize zeroes Goto jump caches, which Detokenize does not keep.
func normalize(dl *ti85family.Dialect, b []byte) []byte {
	toks, err := dl.Decode(b)
	var tail []byte
	if err != nil { // last token is cut short: Detokenize keeps it as raw bytes
		tail = b[toks[len(toks)-1].Offset:]
		toks = toks[:len(toks)-1]
	}
	for i := range toks {
		if toks[i].Code == 0xE1 {
			toks[i].Cache = [2]byte{}
		}
	}
	return append(ti85family.Encode(toks), tail...)
}

var escRe = regexp.MustCompile(`⟦#?[0-9A-F]{2,4}`)

// TestCorpusRoundTrip detokenizes every program and equation and checks
// that tokenizing the text gives back the same bytes.
func TestCorpusRoundTrip(t *testing.T) {
	progs := map[ticalc.Model]int{}
	fails := 0
	escapes := map[string]int{}
	walkCorpus(t, func(it item) {
		dl, ok := it.calc.Codec().(*ti85family.Dialect)
		if !ok {
			return
		}
		for _, v := range it.f.Variables {
			var body []byte
			ti := it.calc.TypeInfo(v.Type)
			switch {
			case ti.Program:
				pi, _ := dl.Inspect(&v)
				if pi.Kind == ti85family.PlainText {
					continue
				}
				body = pi.Body
			case ti.Text && ti.Name == "Equation":
				body = ti85family.InnerData(v.Data)
			default:
				continue
			}
			progs[it.calc.Model()]++
			text, err := dl.Detokenize(body, ticalc.DetokenizeOptions{})
			if err != nil {
				t.Errorf("%s %s: detokenize: %v", it.e, v.Name, err)
				fails++
				continue
			}
			for _, m := range escRe.FindAllString(text, -1) {
				escapes[m+"⟧"]++
			}
			back, err := dl.Tokenize(text)
			want := normalize(dl, body)
			if err != nil || !bytes.Equal(back, want) {
				fails++
				if fails <= 10 {
					i := 0
					for i < len(back) && i < len(want) && back[i] == want[i] {
						i++
					}
					t.Errorf("%s %s: round trip differs at byte %d (err=%v)\n want % X\n got  % X",
						it.e, v.Name, i, err, want[max(0, i-8):min(len(want), i+12)], back[max(0, i-8):min(len(back), i+12)])
				}
			}
		}
	})
	t.Logf("programs/equations per model: %v, round-trip failures: %d", progs, fails)
	type kv struct {
		k string
		v int
	}
	var l []kv
	for k, v := range escapes {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].v > l[j].v })
	for i, x := range l {
		if i < 10 {
			t.Logf("escape %-10s %d", x.k, x.v)
		}
	}
}

// TestCorpusContainer re-marshals every file and checks that strings,
// names and comments survive the charset round trip.
func TestCorpusContainer(t *testing.T) {
	files := map[ticalc.Model]int{}
	walkCorpus(t, func(it item) {
		files[it.calc.Model()]++
		raw, err := it.calc.Marshal(it.f)
		if err != nil {
			t.Errorf("%s: marshal: %v", it.e, err)
			return
		}
		g, err := it.calc.Parse(raw, ticalc.ParseOptions{Strict: true})
		if err != nil {
			t.Errorf("%s: reparse: %v", it.e, err)
			return
		}
		if g.Comment != it.f.Comment || len(g.Variables) != len(it.f.Variables) {
			t.Errorf("%s: header changed", it.e)
			return
		}
		for i, v := range it.f.Variables {
			w := g.Variables[i]
			if w.Name != v.Name || w.Type != v.Type || !bytes.Equal(w.Data, v.Data) {
				t.Errorf("%s %s: variable changed", it.e, v.Name)
			}
			if b, err := ti85family.Charset.Encode(v.Name); err != nil || !bytes.Equal(b, v.RawName) {
				t.Errorf("%s %q: name does not round trip", it.e, v.Name)
			}
			if it.calc.TypeInfo(v.Type).Name == "String" {
				s, _ := it.calc.Codec().Text(&v, ticalc.DetokenizeOptions{})
				if b, err := ti85family.Charset.Encode(s); err != nil || !bytes.Equal(b, ti85family.InnerData(v.Data)) {
					t.Errorf("%s %s: string does not round trip", it.e, v.Name)
				}
			}
		}
	})
	t.Logf("files per model: %v", files)
}
