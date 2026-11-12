// Command tiextract writes every variable found in TI calculator files, zips
// and directories to its own file, plus a .txt for programs and strings.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hjkoskel/hkoTiUtil/internal/cli"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

func main() {
	outDir := flag.String("o", "out", "output `dir`")
	noText := flag.Bool("notext", false, "do not write .txt files")
	cli.Usage("tiextract [-o dir] path...",
		"Extracts each variable to <dir>/<archive path>/<NAME>.<ext> (a standalone\n"+
			"single-variable file) and writes <NAME>.txt for programs, strings and equations.")
	count := 0
	err := cli.Walk(flag.Args(), false, func(it cli.Item) error {
		dir := *outDir
		for _, p := range it.Entry.Path {
			for _, part := range strings.Split(filepath.ToSlash(p), "/") {
				if part == "" || part == "." || part == ".." {
					continue
				}
				dir = filepath.Join(dir, cli.SafeName(part))
			}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for i := range it.File.Variables {
			v := it.File.Variables[i]
			ti := it.Calc.TypeInfo(v.Type)
			raw, err := it.Calc.Marshal(&ticalc.File{Model: it.File.Model, Comment: it.File.Comment, Variables: []ticalc.Variable{v}})
			if err != nil {
				cli.Warn("%s :: %s: %v", it.Entry, v.Name, err)
				continue
			}
			base := filepath.Join(dir, cli.SafeName(v.Name))
			if err := os.WriteFile(base+"."+ti.Ext, raw, 0o644); err != nil {
				return err
			}
			count++
			if *noText || !ti.Text || it.Calc.Codec() == nil || cli.Details(it.Calc, &v).Binary {
				continue
			}
			text, err := it.Calc.Codec().Text(&v, ticalc.DetokenizeOptions{})
			if err != nil {
				cli.Warn("%s :: %s: %v", it.Entry, v.Name, err)
				continue
			}
			if err := os.WriteFile(base+".txt", []byte(text+"\n"), 0o644); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		cli.Fatal(err)
	}
	fmt.Printf("%d variables extracted to %s\n", count, *outDir)
}
