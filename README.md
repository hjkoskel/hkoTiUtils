# hkoTiUtil

Go library and tools for TI calculator variable files. The TI-86 (`.86p`,
`.86g`, `.86s`, …) and TI-85 (`.85p`, `.85g`, …) are supported. The model is
detected from the file. The `ticalc` interfaces are ready for more models.

- Reads files directly, from zip archives, and from zips nested inside zips
- Prints tokenized TI-BASIC programs as readable Unicode text
- Tokenizes text back into `.86p` files; ASCII spellings (`->`, `!=`, `<=`) work too
- Text output round-trips exactly: tokenizing the printed text gives back the same bytes

## Tools

```sh
go install github.com/hjkoskel/hkoTiUtil/cmd/...@latest

tilist  games/                    # list variables in files/zips/dirs
tibasic games/mines.zip           # print programs as text
tibasic -o txt/ games/            # one NAME.txt per program
tiextract -o out/ games/          # every variable as its own file + .txt
titokenize -locked HELLO.txt      # HELLO.txt -> HELLO.86p
titokenize -model TI-85 HELLO.txt # HELLO.txt -> HELLO.85p
tistats download/                 # statistics of a collection
tibrowser games/mines.zip         # graphical browser (Fyne)
```

`tibrowser` opens a file, group file, zip or directory. With more than one
variable the window is split into a variable table (name with a type icon,
type, size; click a column header to sort) and a preview. Programs,
strings and equations are shown as text, pictures as an image plus a Unicode
block drawing, and everything else as a hex dump. The preview text can be
edited, copied (the selection or everything) and pasted, and saved with a
save dialog; pictures can also be saved as PNG. The clipboard only holds
text, so a copied picture is its block drawing.

Building `tibrowser` needs cgo and the OpenGL/X11 development headers (see
the [Fyne prerequisites](https://docs.fyne.io/started/)). The library and the
other tools are pure Go.

Fyne draws its file dialogs inside the window at 90% of the window size, so
enlarge the window for a bigger dialog. On Linux, building with
`go build -tags flatpak ./cmd/tibrowser` uses the desktop's own (resizable)
file dialog through xdg-desktop-portal instead.

## Library

```go
import (
    "github.com/hjkoskel/hkoTiUtil/ticalc"
    _ "github.com/hjkoskel/hkoTiUtil/ti85" // registers the TI-85
    _ "github.com/hjkoskel/hkoTiUtil/ti86" // registers the TI-86
)

f, calc, err := ticalc.Parse(raw, ticalc.ParseOptions{})
for _, v := range f.Variables {
    if calc.TypeInfo(v.Type).Program {
        text, _ := calc.Codec().Text(&v, ticalc.DetokenizeOptions{})
        fmt.Println(text)
    }
    if im, ok := calc.(ticalc.Imager); ok { // pictures
        if img, err := im.Image(&v); err == nil {
            fmt.Print(ticalc.ImageText(img))
        }
    }
}
```

See `HKOTIUTILPLAN.md` for the design, format findings and open work.
