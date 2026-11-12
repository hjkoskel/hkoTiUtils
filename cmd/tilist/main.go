// Command tilist lists the variables in TI calculator files, zips and
// directories.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hjkoskel/hkoTiUtil/internal/cli"
)

type row struct {
	Path       string   `json:"path"`
	Model      string   `json:"model"`
	Comment    string   `json:"comment,omitempty"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Size       int      `json:"size"`
	Kind       string   `json:"kind,omitempty"`
	EditLocked bool     `json:"editLocked,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

func main() {
	asJSON := flag.Bool("json", false, "print JSON lines instead of a table")
	cli.Usage("tilist [-json] path...", "Lists variables in TI calculator files (.86p, .86g, ...), zip archives and directories.")
	enc := json.NewEncoder(os.Stdout)
	err := cli.Walk(flag.Args(), false, func(it cli.Item) error {
		var warns []string
		for _, w := range it.File.Warnings {
			warns = append(warns, w.Error())
		}
		for i := range it.File.Variables {
			v := &it.File.Variables[i]
			d := cli.Details(it.Calc, v)
			r := row{Path: it.Entry.String(), Model: string(it.Calc.Model()), Comment: it.File.Comment,
				Name: v.Name, Type: it.Calc.TypeInfo(v.Type).Name, Size: len(v.Data),
				Kind: d.Kind, EditLocked: d.EditLocked, Warnings: warns}
			if *asJSON {
				enc.Encode(r)
				continue
			}
			extra := d.Kind
			if d.EditLocked {
				extra += ", edit-locked"
			}
			fmt.Printf("%-50s %-6s %-8s %-14s %6d  %s\n", r.Path, r.Model, r.Name, r.Type, r.Size, strings.TrimPrefix(extra, ", "))
		}
		if !*asJSON {
			for _, w := range warns {
				fmt.Printf("    ! %s\n", w)
			}
		}
		return nil
	})
	if err != nil {
		cli.Fatal(err)
	}
}
