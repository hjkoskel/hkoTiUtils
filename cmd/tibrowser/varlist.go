package main

import (
	"cmp"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hjkoskel/hkoTiUtil/internal/cli"
)

// column is one column of the variable table.
type column struct {
	title string
	width float32
	value func(it item) string
	less  func(a, b item) int
}

func typeName(it item) string { return it.calc.TypeInfo(it.v.Type).Name }

var (
	colName = column{"Name", 150, func(it item) string { return it.v.Name },
		func(a, b item) int { return cmp.Compare(strings.ToLower(a.v.Name), strings.ToLower(b.v.Name)) }}
	colType = column{"Type", 210, func(it item) string {
		if k := cli.Details(it.calc, &it.v).Kind; k != "" {
			return typeName(it) + " (" + k + ")"
		}
		return typeName(it)
	}, func(a, b item) int { return cmp.Compare(typeName(a), typeName(b)) }}
	colSize = column{"Size", 90, func(it item) string { return strconv.Itoa(len(it.v.Data)) },
		func(a, b item) int { return cmp.Compare(len(a.v.Data), len(b.v.Data)) }}
	colSource = column{"Source", 220, func(it item) string { return filepath.Base(it.src) },
		func(a, b item) int { return cmp.Compare(a.src, b.src) }}
)

// typeIcon picks an icon for a variable from its type and program kind.
// The theme's File* icons all look alike, so distinct shapes are used.
func typeIcon(it item) fyne.Resource {
	t := typeName(it)
	switch {
	case t == "Program":
		switch cli.Details(it.calc, &it.v).Kind {
		case "assembly":
			return theme.SettingsIcon() // machine code
		case "plain text":
			return theme.DocumentCreateIcon() // editable source
		}
		return theme.DocumentIcon()
	case t == "Picture":
		return theme.MediaPhotoIcon()
	case t == "String":
		return theme.MenuIcon()
	case t == "Equation":
		return theme.ContentAddIcon()
	case strings.Contains(t, "Matrix"):
		return theme.GridIcon()
	case strings.Contains(t, "List"), strings.Contains(t, "Vector"):
		return theme.ListIcon()
	case strings.HasPrefix(t, "Real"), strings.HasPrefix(t, "Complex"):
		return theme.ComputerIcon()
	case strings.Contains(t, "GDB"):
		return theme.ColorPaletteIcon()
	case strings.Contains(t, "Window"), strings.Contains(t, "Range"):
		return theme.ViewFullScreenIcon()
	case t == "Directory":
		return theme.FolderIcon()
	case strings.Contains(t, "Backup"):
		return theme.StorageIcon()
	}
	return theme.QuestionIcon()
}

// varTable is the sortable variable list on the left of a split window.
type varTable struct {
	b       *browser
	cols    []column
	order   []int // displayed row → index in b.items
	sortCol int   // -1: file order
	desc    bool
	sel     int // selected item index, -1 if none
	table   *widget.Table
}

func newVarTable(b *browser) *varTable {
	t := &varTable{b: b, cols: []column{colName, colType, colSize}, sortCol: -1, sel: -1}
	if b.items[0].src != b.items[len(b.items)-1].src {
		t.cols = append(t.cols, colSource)
	}
	t.order = make([]int, len(b.items))
	for i := range t.order {
		t.order[i] = i
	}
	t.table = widget.NewTableWithHeaders(
		func() (int, int) { return len(t.order), len(t.cols) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(nil, nil, widget.NewIcon(nil), nil, l)
		},
		t.updateCell,
	)
	t.table.ShowHeaderColumn = false
	t.table.CreateHeader = func() fyne.CanvasObject {
		h := widget.NewButton("", nil)
		h.Alignment = widget.ButtonAlignLeading
		h.Importance = widget.LowImportance
		return h
	}
	t.table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		h := o.(*widget.Button)
		if id.Col < 0 {
			return
		}
		h.SetText(t.cols[id.Col].title)
		h.SetIcon(nil)
		if id.Col == t.sortCol {
			h.SetIcon(theme.MenuDropDownIcon())
			if t.desc {
				h.SetIcon(theme.MenuDropUpIcon())
			}
		}
		col := id.Col
		h.OnTapped = func() { t.sortBy(col) }
	}
	for i, c := range t.cols {
		t.table.SetColumnWidth(i, c.width)
	}
	t.table.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(t.order) {
			return
		}
		if t.order[id.Row] != t.sel {
			t.sel = t.order[id.Row]
			b.show(t.sel)
		}
		t.table.Refresh() // highlight the whole row
	}
	return t
}

func (t *varTable) updateCell(id widget.TableCellID, o fyne.CanvasObject) {
	idx := t.order[id.Row]
	it := t.b.items[idx]
	c := o.(*fyne.Container)
	l, icon := c.Objects[0].(*widget.Label), c.Objects[1].(*widget.Icon)
	l.SetText(t.cols[id.Col].value(it))
	l.TextStyle.Bold = idx == t.sel
	l.Refresh()
	if id.Col == 0 {
		icon.SetResource(typeIcon(it))
		icon.Show()
	} else {
		icon.Hide()
	}
}

// sortBy sorts by column col; sorting by the same column again reverses
// the order. Ties keep the file order.
func (t *varTable) sortBy(col int) {
	if t.sortCol == col {
		t.desc = !t.desc
	} else {
		t.sortCol, t.desc = col, false
	}
	less := t.cols[col].less
	slices.SortStableFunc(t.order, func(a, b int) int {
		r := less(t.b.items[a], t.b.items[b])
		if r == 0 {
			r = cmp.Compare(a, b)
		} else if t.desc {
			r = -r
		}
		return r
	})
	if row := slices.Index(t.order, t.sel); row >= 0 {
		t.selectRow(row) // the selection follows the item
	}
	t.table.Refresh()
}

// selectRow selects the item shown on row.
func (t *varTable) selectRow(row int) {
	t.table.Select(widget.TableCellID{Row: row, Col: 0})
}
