package tiarchive_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"

	"github.com/hjkoskel/hkoTiUtil/tiarchive"
)

func makeZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, data := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write(data)
	}
	w.Close()
	return buf.Bytes()
}

func TestNestedZip(t *testing.T) {
	ti := []byte("**TI86**\x1a\x0a\x00 rest of file")
	inner := makeZip(t, map[string][]byte{"GAME.86P": ti, "readme.txt": []byte("hi")})
	outer := makeZip(t, map[string][]byte{"inner.zip": inner, "X.86G": ti})

	var got []string
	err := tiarchive.WalkBytes("outer.zip", outer, tiarchive.Options{}, func(e tiarchive.Entry) error {
		got = append(got, e.String())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"outer.zip::inner.zip::GAME.86P": true, "outer.zip::X.86G": true}
	if len(got) != 2 || !want[got[0]] || !want[got[1]] {
		t.Errorf("got %v", got)
	}
}

func TestDepthLimitAndStop(t *testing.T) {
	ti := []byte("**TI86**\x1a\x0a\x00")
	z := makeZip(t, map[string][]byte{"A.86P": ti})
	for i := 0; i < 5; i++ {
		z = makeZip(t, map[string][]byte{"n.zip": z})
	}
	var problems int
	n := 0
	tiarchive.WalkBytes("deep.zip", z, tiarchive.Options{MaxDepth: 3, OnError: func([]string, error) { problems++ }}, func(tiarchive.Entry) error {
		n++
		return nil
	})
	if n != 0 || problems != 1 {
		t.Errorf("found %d, problems %d", n, problems)
	}

	stop := errors.New("stop")
	err := tiarchive.WalkBytes("a.zip", makeZip(t, map[string][]byte{"A.86P": ti, "B.86P": ti}), tiarchive.Options{}, func(tiarchive.Entry) error { return stop })
	if err != stop {
		t.Errorf("callback error not returned: %v", err)
	}
}
