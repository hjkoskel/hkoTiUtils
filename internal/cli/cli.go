// Package cli holds helpers shared by the command line tools.
package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hjkoskel/hkoTiUtil/tiarchive"
	"github.com/hjkoskel/hkoTiUtil/ticalc"

	// Register supported calculators.
	_ "github.com/hjkoskel/hkoTiUtil/ti85"
	_ "github.com/hjkoskel/hkoTiUtil/ti86"
)

// Item is one parsed calculator file found while walking.
type Item struct {
	Entry tiarchive.Entry
	File  *ticalc.File
	Calc  ticalc.Calculator
}

// Walk parses every calculator file under paths (files, directories, zips,
// nested zips). Problems are reported to stderr and do not stop the walk.
func Walk(paths []string, strict bool, fn func(Item) error) error {
	opt := tiarchive.Options{OnError: func(p []string, err error) {
		Warn("%s: %v", strings.Join(p, "::"), err)
	}}
	for _, root := range paths {
		err := tiarchive.Walk(root, opt, func(e tiarchive.Entry) error {
			f, c, err := ticalc.Parse(e.Data, ticalc.ParseOptions{Strict: strict})
			if err != nil {
				Warn("%s: %v", e, err)
				return nil
			}
			return fn(Item{Entry: e, File: f, Calc: c})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Details returns calculator-specific details when available.
func Details(c ticalc.Calculator, v *ticalc.Variable) ticalc.Details {
	if d, ok := c.(ticalc.Describer); ok {
		return d.Details(v)
	}
	return ticalc.Details{}
}

// Warn prints a message to stderr.
func Warn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", a...)
}

// Fatal prints an error and exits.
func Fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

// Usage installs a usage message and requires at least one argument.
func Usage(synopsis, about string) {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s\n\n%s\n\nflags:\n", synopsis, about)
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
}

// SafeName turns a calculator or archive name into a file name component.
func SafeName(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r < 0x20, strings.ContainsRune(`/\:*?"<>|⟦⟧`, r):
			return '_'
		}
		return r
	}, s)
	s = strings.Trim(s, ". ")
	if s == "" {
		return "_"
	}
	return s
}
