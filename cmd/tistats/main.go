// Command tistats prints statistics about a collection of TI calculator
// files: models, variable types, program kinds, warnings and how many
// escapes (unknown tokens or glyphs) the text output still contains.
package main

import (
	"flag"
	"fmt"
	"regexp"
	"sort"

	"github.com/hjkoskel/hkoTiUtil/internal/cli"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

type counter map[string]int

func (c counter) print(title string, limit int) {
	type kv struct {
		k string
		v int
	}
	var l []kv
	for k, v := range c {
		l = append(l, kv{k, v})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].v > l[j].v || l[i].v == l[j].v && l[i].k < l[j].k })
	fmt.Printf("\n%s\n", title)
	for i, x := range l {
		if limit > 0 && i >= limit {
			fmt.Printf("  … %d more\n", len(l)-limit)
			break
		}
		fmt.Printf("  %7d  %s\n", x.v, x.k)
	}
}

var numRe = regexp.MustCompile(`[0-9A-F]{4}\b|\d+|"[^"]*"`)

var escRe = regexp.MustCompile(`⟦#?[0-9A-F]{2,4}`)

func main() {
	limit := flag.Int("n", 20, "rows per table (0 = all)")
	cli.Usage("tistats [-n rows] path...", "Statistics of TI calculator files in files, zips and directories.")
	models, types, kinds, warns, escapes := counter{}, counter{}, counter{}, counter{}, counter{}
	files := 0
	err := cli.Walk(flag.Args(), false, func(it cli.Item) error {
		files++
		models[string(it.Calc.Model())]++
		for _, w := range it.File.Warnings {
			warns[numRe.ReplaceAllString(w.Error(), "N")]++
		}
		for i := range it.File.Variables {
			v := &it.File.Variables[i]
			ti := it.Calc.TypeInfo(v.Type)
			types[ti.Name]++
			d := cli.Details(it.Calc, v)
			if ti.Program {
				k := d.Kind
				if d.EditLocked {
					k += ", edit-locked"
				}
				kinds[k]++
			}
			if !ti.Text || d.Binary || it.Calc.Codec() == nil {
				continue
			}
			text, err := it.Calc.Codec().Text(v, ticalc.DetokenizeOptions{})
			if err != nil {
				warns["text: "+err.Error()]++
				continue
			}
			for _, m := range escRe.FindAllString(text, -1) {
				escapes[ti.Name+" "+m+"⟧"]++
			}
		}
		return nil
	})
	if err != nil {
		cli.Fatal(err)
	}
	fmt.Printf("%d calculator files\n", files)
	models.print("Models", *limit)
	types.print("Variable types", *limit)
	kinds.print("Program kinds", *limit)
	warns.print("Warnings", *limit)
	escapes.print("Escapes in text output (unknown tokens / glyphs)", *limit)
}
