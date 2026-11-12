// Command titokenize turns TI-BASIC source text into a calculator program
// file (e.g. .86p).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/hjkoskel/hkoTiUtil/internal/cli"
	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

func main() {
	model := flag.String("model", string(ticalc.ModelTI86), "calculator model")
	name := flag.String("name", "", "program name (default: input file name)")
	out := flag.String("o", "", "output file (default: NAME.<ext> next to the input)")
	comment := flag.String("comment", "Created by hkoTiUtil", "file comment")
	locked := flag.Bool("locked", false, "mark the program edit-locked")
	cli.Usage("titokenize [flags] program.txt",
		"Tokenizes TI-BASIC source. Unicode (→ ≠ ≤ ≥ ▶) and ASCII (-> != <= >= >)\n"+
			"spellings are both accepted. Use - to read standard input.")

	c, ok := ticalc.Lookup(ticalc.Model(*model))
	if !ok || c.Codec() == nil {
		cli.Fatal(fmt.Errorf("no tokenizer for model %q (have %v)", *model, ticalc.Models()))
	}
	in := flag.Arg(0)
	var src []byte
	var err error
	if in == "-" {
		src, err = io.ReadAll(os.Stdin)
	} else {
		src, err = os.ReadFile(in)
	}
	if err != nil {
		cli.Fatal(err)
	}
	if !utf8.Valid(src) {
		cli.Fatal(fmt.Errorf("%s: input is not UTF-8", in))
	}
	text := strings.TrimRight(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	if *name == "" {
		if in == "-" {
			cli.Fatal(fmt.Errorf("-name is required when reading standard input"))
		}
		*name = strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
	}
	v, err := c.Codec().NewProgram(*name, text, ticalc.TokenizeOptions{EditLocked: *locked})
	if err != nil {
		cli.Fatal(fmt.Errorf("%s: %w", in, err))
	}
	raw, err := c.Marshal(&ticalc.File{Model: c.Model(), Comment: *comment, Variables: []ticalc.Variable{*v}})
	if err != nil {
		cli.Fatal(err)
	}
	if *out == "" {
		*out = filepath.Join(filepath.Dir(in), cli.SafeName(*name)+"."+c.TypeInfo(v.Type).Ext)
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		cli.Fatal(err)
	}
	fmt.Printf("%s: %d bytes of tokens\n", *out, len(v.Data)-2)
}
