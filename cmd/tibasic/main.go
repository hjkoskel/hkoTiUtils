// Command tibasic prints TI-BASIC programs (and optionally strings and
// equations) as Unicode text.
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
	outDir := flag.String("o", "", "write each program to `dir`/NAME.txt instead of stdout")
	all := flag.Bool("all", false, "also print strings and equations")
	asm := flag.Bool("asm", false, "include assembly programs as hex dumps")
	strict := flag.Bool("strict", false, "fail on unknown tokens and damaged files")
	quiet := flag.Bool("q", false, "no header lines (stdout only)")
	cli.Usage("tibasic [flags] path...", "Prints TI-BASIC programs from TI calculator files, zips and directories as text.")

	written := map[string]int{}
	err := cli.Walk(flag.Args(), *strict, func(it cli.Item) error {
		codec := it.Calc.Codec()
		if codec == nil {
			cli.Warn("%s: no program decoder for %s", it.Entry, it.Calc.Model())
			return nil
		}
		for i := range it.File.Variables {
			v := &it.File.Variables[i]
			ti := it.Calc.TypeInfo(v.Type)
			if !ti.Text || (!ti.Program && !*all) {
				continue
			}
			d := cli.Details(it.Calc, v)
			if d.Binary && !*asm {
				if !*quiet && *outDir == "" {
					fmt.Printf("=== %s :: %s (%s, %s, %d bytes, skipped; use -asm)\n\n", it.Entry, v.Name, ti.Name, d.Kind, len(v.Data))
				}
				continue
			}
			text, err := codec.Text(v, ticalc.DetokenizeOptions{Strict: *strict})
			if err != nil {
				cli.Warn("%s :: %s: %v", it.Entry, v.Name, err)
				continue
			}
			if *outDir != "" {
				name := cli.SafeName(v.Name)
				if n := written[name]; n > 0 {
					name = fmt.Sprintf("%s_%d", name, n+1)
				}
				written[cli.SafeName(v.Name)]++
				if err := os.MkdirAll(*outDir, 0o755); err != nil {
					return err
				}
				p := filepath.Join(*outDir, name+".txt")
				if err := os.WriteFile(p, []byte(text+"\n"), 0o644); err != nil {
					return err
				}
				fmt.Println(p)
				continue
			}
			if !*quiet {
				info := []string{ti.Name}
				if d.Kind != "" {
					info = append(info, d.Kind)
				}
				if d.EditLocked {
					info = append(info, "edit-locked")
				}
				fmt.Printf("=== %s :: %s (%s)\n", it.Entry, v.Name, strings.Join(info, ", "))
			}
			fmt.Println(text)
			fmt.Println()
		}
		return nil
	})
	if err != nil {
		cli.Fatal(err)
	}
}
