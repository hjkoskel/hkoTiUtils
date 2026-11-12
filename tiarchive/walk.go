// Package tiarchive finds calculator variable files in directories, plain
// files and zip archives (including zips nested inside zips).
package tiarchive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// Entry is one candidate file found while walking.
type Entry struct {
	// Path is the location chain: the file on disk followed by the names
	// inside each nested archive, e.g. ["games/3games.zip", "inner.zip", "GAME.86P"].
	Path []string
	Data []byte
}

// Name is the base name of the entry.
func (e Entry) Name() string { return e.Path[len(e.Path)-1] }

// String joins the path chain with "::" between archive levels.
func (e Entry) String() string { return strings.Join(e.Path, "::") }

// Options control walking.
type Options struct {
	// MaxDepth limits zip nesting (default 4).
	MaxDepth int
	// MaxSize skips entries larger than this many bytes (default 16 MiB).
	MaxSize int64
	// All passes every file to the callback. By default only files that
	// start with a "**TIxx**" signature are passed.
	All bool
	// OnError receives non-fatal problems (unreadable zip entry,
	// unsupported compression, ...). When nil they are ignored.
	OnError func(path []string, err error)
}

// ErrUnsupportedCompression is reported for zip entries using old PKZIP
// methods (Shrink, Implode, ...) that archive/zip cannot read.
var ErrUnsupportedCompression = errors.New("unsupported zip compression method")

func (o *Options) defaults() {
	if o.MaxDepth == 0 {
		o.MaxDepth = 4
	}
	if o.MaxSize == 0 {
		o.MaxSize = 16 << 20
	}
}

// Walk visits root, which may be a directory, a zip file or a single file.
// Returning an error from fn stops the walk with that error.
func Walk(root string, opt Options, fn func(Entry) error) error {
	opt.defaults()
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			opt.report([]string{p}, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			opt.report([]string{p}, err)
			return nil
		}
		if info.Size() > opt.MaxSize {
			opt.report([]string{p}, fmt.Errorf("skipped: %d bytes is over size limit", info.Size()))
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			opt.report([]string{p}, err)
			return nil
		}
		return walkData([]string{p}, data, 0, &opt, fn)
	})
}

// WalkBytes visits an in-memory file (zip or single variable file).
func WalkBytes(name string, data []byte, opt Options, fn func(Entry) error) error {
	opt.defaults()
	return walkData([]string{name}, data, 0, &opt, fn)
}

func (o *Options) report(path []string, err error) {
	if o.OnError != nil {
		o.OnError(path, err)
	}
}

func isZip(data []byte) bool {
	return len(data) >= 4 && bytes.Equal(data[:4], []byte("PK\x03\x04"))
}

func walkData(path []string, data []byte, depth int, opt *Options, fn func(Entry) error) error {
	if isZip(data) {
		if depth >= opt.MaxDepth {
			opt.report(path, fmt.Errorf("zip nesting deeper than %d", opt.MaxDepth))
			return nil
		}
		return walkZip(path, data, depth, opt, fn)
	}
	if opt.All || ticalc.Signature(data) != "" {
		return fn(Entry{Path: path, Data: data})
	}
	return nil
}

func walkZip(path []string, data []byte, depth int, opt *Options, fn func(Entry) error) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		opt.report(path, fmt.Errorf("reading zip: %w", err))
		return nil
	}
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			continue
		}
		sub := append(append([]string(nil), path...), zf.Name)
		if zf.Method != zip.Store && zf.Method != zip.Deflate {
			opt.report(sub, fmt.Errorf("%w %d", ErrUnsupportedCompression, zf.Method))
			continue
		}
		if int64(zf.UncompressedSize64) > opt.MaxSize {
			opt.report(sub, fmt.Errorf("skipped: %d bytes is over size limit", zf.UncompressedSize64))
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			opt.report(sub, err)
			continue
		}
		// LimitReader guards against entries lying about their size.
		b, err := io.ReadAll(io.LimitReader(rc, opt.MaxSize+1))
		rc.Close()
		if err != nil {
			opt.report(sub, err)
			continue
		}
		if int64(len(b)) > opt.MaxSize {
			opt.report(sub, fmt.Errorf("skipped: over size limit"))
			continue
		}
		if err := walkData(sub, b, depth+1, opt, fn); err != nil {
			return err
		}
	}
	return nil
}
