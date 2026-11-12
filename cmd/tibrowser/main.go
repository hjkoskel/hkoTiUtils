// Command tibrowser is a graphical browser for TI calculator files.
//
//	tibrowser [file]
//
// The file can be a single-variable file, a group file (.86g/.85g), a zip
// or a directory. With more than one variable the window is split: the
// variable list is on the left and the preview on the right.
//
// Programs, strings and equations are shown as text, pictures as an image
// together with a Unicode block drawing of it, and other variables as a hex
// dump. The preview text can be edited, copied and pasted, and saved as a
// text file. Pictures can also be saved as PNG.
package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hjkoskel/hkoTiUtil/internal/cli"
	"github.com/hjkoskel/hkoTiUtil/tiarchive"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// item is one variable found in the opened file.
type item struct {
	src      string // path inside the opened file, e.g. "games.zip::MINES.86g"
	calc     ticalc.Calculator
	v        ticalc.Variable
	warnings []error // container warnings of the file the variable came from
}

// preview is what the right-hand side shows for an item.
type preview struct {
	info string
	text string
	img  image.Image
}

// load reads every variable from path (file, zip or directory).
func load(path string) ([]item, error) {
	var items []item
	var problems []string
	opt := tiarchive.Options{OnError: func(p []string, err error) {
		problems = append(problems, fmt.Sprintf("%s: %v", strings.Join(p, "::"), err))
	}}
	err := tiarchive.Walk(path, opt, func(e tiarchive.Entry) error {
		f, c, err := ticalc.Parse(e.Data, ticalc.ParseOptions{})
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", e, err))
			return nil
		}
		for _, v := range f.Variables {
			items = append(items, item{src: e.String(), calc: c, v: v, warnings: f.Warnings})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		msg := "no calculator variables found"
		if len(problems) > 0 {
			msg += ":\n" + strings.Join(problems, "\n")
		}
		return nil, errors.New(msg)
	}
	return items, nil
}

// render builds the preview of a variable.
func render(it item) preview {
	ti := it.calc.TypeInfo(it.v.Type)
	info := fmt.Sprintf("%s — %s, %s, %d bytes\n%s", it.v.Name, ti.Name, it.calc.Model(), len(it.v.Data), it.src)
	if d := cli.Details(it.calc, &it.v); d.Kind != "" {
		info += "\n" + d.Kind
		if d.EditLocked {
			info += ", edit-locked"
		}
	}
	for _, w := range it.warnings {
		info += "\nwarning: " + w.Error()
	}
	p := preview{info: info}

	if im, ok := it.calc.(ticalc.Imager); ok {
		img, err := im.Image(&it.v)
		if !errors.Is(err, ticalc.ErrNotImage) {
			if err != nil {
				p.info += "\nwarning: " + err.Error()
			}
			if img != nil {
				p.img = img
				p.text = ticalc.ImageText(img)
				return p
			}
		}
	}
	if ti.Text || ti.Program {
		if c := it.calc.Codec(); c != nil {
			text, err := c.Text(&it.v, ticalc.DetokenizeOptions{})
			if err == nil {
				p.text = text
				return p
			}
			p.info += "\n" + err.Error()
		}
	}
	p.text = hex.Dump(it.v.Data)
	return p
}

// browser is the main window state.
type browser struct {
	app     fyne.App
	win     fyne.Window
	dir     string // directory of the opened file, start point for dialogs
	items   []item
	current preview
	name    string // base name for saved files

	info   *widget.Label
	image  *canvas.Image
	imgBox *fyne.Container
	editor *widget.Entry
	right  *fyne.Container // info, picture and editor
	vars   *varTable       // variable list; nil for a single variable
}

func newBrowser(a fyne.App) *browser {
	b := &browser{app: a, win: a.NewWindow("tibrowser")}

	b.info = widget.NewLabel("")
	b.info.Truncation = fyne.TextTruncateEllipsis
	b.image = canvas.NewImageFromImage(nil)
	b.image.FillMode = canvas.ImageFillContain
	b.image.ScaleMode = canvas.ImageScalePixels
	b.image.SetMinSize(fyne.NewSize(3*128, 3*64))
	b.imgBox = container.NewPadded(b.image)
	b.imgBox.Hide()
	b.editor = widget.NewMultiLineEntry()
	b.editor.TextStyle = fyne.TextStyle{Monospace: true}
	b.editor.Wrapping = fyne.TextWrapOff

	b.win.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("File",
			&fyne.MenuItem{Label: "Open…", Action: b.openDialog, Icon: theme.FolderOpenIcon(),
				Shortcut: &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierShortcutDefault}},
			&fyne.MenuItem{Label: "Save text…", Action: b.saveText, Icon: theme.DocumentSaveIcon(),
				Shortcut: &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault}},
			&fyne.MenuItem{Label: "Save picture as PNG…", Action: b.savePNG},
		),
		fyne.NewMenu("Edit",
			&fyne.MenuItem{Label: "Copy (selection or all)", Action: b.copy, Icon: theme.ContentCopyIcon(),
				Shortcut: &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift}},
			&fyne.MenuItem{Label: "Paste", Action: b.paste, Icon: theme.ContentPasteIcon()},
		),
	))
	b.win.Resize(fyne.NewSize(1000, 700))
	b.win.SetContent(b.emptyContent())
	return b
}

func (b *browser) emptyContent() fyne.CanvasObject {
	return container.NewCenter(widget.NewButtonWithIcon("Open a TI file…", theme.FolderOpenIcon(), b.openDialog))
}

func (b *browser) toolbar() *widget.Toolbar {
	return widget.NewToolbar(
		widget.NewToolbarAction(theme.FolderOpenIcon(), b.openDialog),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), b.saveText),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ContentCopyIcon(), b.copy),
		widget.NewToolbarAction(theme.ContentPasteIcon(), b.paste),
	)
}

// open loads path and rebuilds the window content.
func (b *browser) open(path string) {
	items, err := load(path)
	if err != nil {
		dialog.ShowError(fmt.Errorf("%s: %w", path, err), b.win)
		return
	}
	b.items = items
	b.dir = path
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		b.dir = filepath.Dir(path)
	}
	b.win.SetTitle("tibrowser — " + filepath.Base(path))

	b.right = container.NewBorder(container.NewVBox(b.info, b.imgBox), nil, nil, nil, b.editor)
	var body fyne.CanvasObject = b.right
	b.vars = nil
	if len(items) > 1 {
		b.vars = newVarTable(b)
		split := container.NewHSplit(b.vars.table, b.right)
		split.Offset = 0.5
		body = split
		defer b.vars.selectRow(0)
	} else {
		b.show(0)
	}
	b.win.SetContent(container.NewBorder(b.toolbar(), nil, nil, nil, body))
}

// show puts item i into the preview.
func (b *browser) show(i int) {
	it := b.items[i]
	b.current = render(it)
	b.name = cli.SafeName(it.v.Name)
	b.info.SetText(b.current.info)
	if b.current.img != nil {
		b.image.Image = b.current.img
		b.image.Refresh()
		b.imgBox.Show()
	} else {
		b.imgBox.Hide()
	}
	b.editor.SetText(b.current.text)
	b.right.Refresh() // info and picture change the layout
}

func (b *browser) openDialog() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, b.win)
			return
		}
		if r == nil {
			return
		}
		r.Close()
		b.open(r.URI().Path())
	}, b.win)
	b.showDialog(d)
}

// showDialog shows a file dialog in the directory of the opened file,
// sized to most of the window. Fyne draws dialogs inside the window, so a
// bigger window gives a bigger dialog.
func (b *browser) showDialog(d *dialog.FileDialog) {
	if b.dir != "" {
		if l, err := storage.ListerForURI(storage.NewFileURI(b.dir)); err == nil {
			d.SetLocation(l)
		}
	}
	d.Show()
	sz := b.win.Canvas().Size()
	d.Resize(fyne.NewSize(sz.Width*0.9, sz.Height*0.9))
	// The popup keeps the position centred for its old size; refreshing the
	// overlay lays it out again.
	if o := b.win.Canvas().Overlays().Top(); o != nil {
		o.Refresh()
	}
}

// save shows a save dialog and writes with fn.
func (b *browser) save(name string, ext string, fn func(io.Writer) error) {
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, b.win)
			return
		}
		if w == nil {
			return
		}
		err = fn(w)
		if cerr := w.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			dialog.ShowError(err, b.win)
		}
	}, b.win)
	d.SetFileName(name + ext)
	d.SetFilter(storage.NewExtensionFileFilter([]string{ext}))
	b.showDialog(d)
}

// saveText saves the editor content, including any edits and pastes.
func (b *browser) saveText() {
	if b.items == nil {
		return
	}
	text := b.editor.Text
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	b.save(b.name, ".txt", func(w io.Writer) error {
		_, err := io.WriteString(w, text)
		return err
	})
}

func (b *browser) savePNG() {
	img := b.current.img
	if img == nil {
		dialog.ShowInformation("Save picture", "The selected variable is not a picture.", b.win)
		return
	}
	b.save(b.name, ".png", func(w io.Writer) error { return png.Encode(w, img) })
}

// copy puts the selected text, or all of it, on the clipboard. Pictures are
// copied as their block drawing: the clipboard only holds text.
func (b *browser) copy() {
	s := b.editor.SelectedText()
	if s == "" {
		s = b.editor.Text
	}
	b.app.Clipboard().SetContent(s)
}

func (b *browser) paste() {
	b.win.Canvas().Focus(b.editor)
	b.editor.TypedShortcut(&fyne.ShortcutPaste{Clipboard: b.app.Clipboard()})
}

func main() {
	a := app.NewWithID("io.github.hjkoskel.tibrowser")
	b := newBrowser(a)
	if len(os.Args) > 1 {
		b.open(os.Args[1])
	}
	b.win.ShowAndRun()
}
