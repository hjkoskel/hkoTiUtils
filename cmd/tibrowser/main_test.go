package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func newTestBrowser(t *testing.T) *browser {
	t.Helper()
	a := test.NewApp()
	t.Cleanup(a.Quit)
	return newBrowser(a)
}

// A group file splits the window and the first variable is shown.
func TestOpenGroup(t *testing.T) {
	b := newTestBrowser(t)
	b.open("../../testdata/AUTOPELI.86G")
	if len(b.items) < 2 {
		t.Fatalf("%d items", len(b.items))
	}
	border := b.win.Content().(*fyne.Container)
	var split *container.Split
	for _, o := range border.Objects {
		if s, ok := o.(*container.Split); ok {
			split = s
		}
	}
	if split == nil {
		t.Fatal("window is not split")
	}
	if _, ok := split.Leading.(*widget.Table); !ok {
		t.Errorf("left side is %T", split.Leading)
	}
	if b.editor.Text == "" || !strings.Contains(b.info.Text, b.items[0].v.Name) {
		t.Errorf("first item not shown: %q", b.info.Text)
	}
}

// A single-variable file is shown without a list.
func TestOpenSingle(t *testing.T) {
	b := newTestBrowser(t)
	b.open("../../testdata/MINES.86P")
	if len(b.items) != 1 {
		t.Fatalf("%d items", len(b.items))
	}
	if b.editor.Text == "" || strings.HasPrefix(b.editor.Text, "00000000  ") {
		t.Errorf("program text missing: %q", b.editor.Text)
	}
	if b.imgBox.Visible() {
		t.Error("image shown for a program")
	}
}

// Pictures show an image and a block drawing; copy puts it on the clipboard.
func TestPictureAndCopy(t *testing.T) {
	b := newTestBrowser(t)
	b.open("../../testdata/TT3DPIC.86i")
	if b.current.img == nil || !b.imgBox.Visible() {
		t.Fatal("picture not shown")
	}
	if !strings.ContainsAny(b.editor.Text, "▀▄█") {
		t.Error("no block drawing")
	}
	b.copy()
	if got := b.app.Clipboard().Content(); got != b.editor.Text {
		t.Errorf("clipboard has %d bytes, editor %d", len(got), len(b.editor.Text))
	}
	b.app.Clipboard().SetContent("pasted")
	b.editor.SetText("")
	b.paste()
	if b.editor.Text != "pasted" {
		t.Errorf("paste gave %q", b.editor.Text)
	}
}

// Non-text variables fall back to a hex dump.
func TestHexDump(t *testing.T) {
	b := newTestBrowser(t)
	b.open("../../testdata/A.86n")
	if !strings.HasPrefix(b.editor.Text, "00000000  ") {
		t.Errorf("not a hex dump: %q", b.editor.Text)
	}
}

// Sorting by type groups the variables; sorting again reverses; the
// selected variable stays selected.
func TestSort(t *testing.T) {
	b := newTestBrowser(t)
	b.open("../../testdata/AUTOPELI.86G") // alkuauto, AUTOPELI (programs), A, B, C (reals)
	vt := b.vars
	names := func() string {
		var s []string
		for _, i := range vt.order {
			s = append(s, b.items[i].v.Name)
		}
		return strings.Join(s, " ")
	}
	sel := vt.sel
	vt.sortBy(1) // Type
	if got := names(); got != "alkuauto AUTOPELI A B C" {
		t.Errorf("by type: %s", got)
	}
	vt.sortBy(1)
	if got := names(); got != "A B C alkuauto AUTOPELI" {
		t.Errorf("by type, descending: %s", got)
	}
	vt.sortBy(0) // Name, case-insensitive
	if got := names(); got != "A alkuauto AUTOPELI B C" {
		t.Errorf("by name: %s", got)
	}
	if vt.sel != sel || !strings.Contains(b.info.Text, b.items[sel].v.Name) {
		t.Errorf("selection lost: %d -> %d", sel, vt.sel)
	}
	vt.selectRow(3) // B
	if !strings.HasPrefix(b.info.Text, "B ") {
		t.Errorf("row 3 shows %q", b.info.Text)
	}
}

func TestTypeIcons(t *testing.T) {
	b := newTestBrowser(t)
	b.open("../../download/basic/ti86/games/3dtictactoe.zip")
	if b.vars == nil {
		t.Skip("no corpus")
	}
	seen := map[string]string{}
	for _, it := range b.items {
		seen[typeName(it)] = typeIcon(it).Name()
	}
	if seen["Program"] == seen["Picture"] {
		t.Errorf("same icon for program and picture: %v", seen)
	}
	if len(b.vars.cols) != 4 {
		t.Errorf("want a Source column for several files, got %d columns", len(b.vars.cols))
	}
}
