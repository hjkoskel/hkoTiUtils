// Package ticalc holds the calculator-independent parts of hkoTiUtil: the
// common File/Variable model, the Calculator interface that every supported
// calculator model implements, and a registry used to auto-detect the model
// of a file from its signature.
//
// Adding a new calculator (TI-85, TI-83, ...) means creating a package that
// implements Calculator (and optionally ProgramCodec) and calling Register
// from its init function. See HKOTIUTILPLAN.md, section "Adding a calculator".
package ticalc

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Model identifies a calculator model, e.g. "TI-86".
type Model string

const (
	ModelTI85 Model = "TI-85"
	ModelTI86 Model = "TI-86"
)

// VarType is a calculator-specific variable type code. Its meaning depends on
// the Calculator; use Calculator.TypeInfo to describe it.
type VarType byte

// TypeInfo describes a variable type of one calculator model.
type TypeInfo struct {
	Name string // human readable, e.g. "Program"
	Ext  string // file extension for a single variable, without dot, e.g. "86p"
	// Text is true when the variable can be rendered as text with
	// ProgramCodec.Text (programs, strings, equations).
	Text bool
	// Program is true for program variables.
	Program bool
}

// Variable is one variable stored in a calculator file.
type Variable struct {
	Type    VarType
	Name    string // decoded to UTF-8 with the calculator charset
	RawName []byte // name bytes exactly as stored
	Data    []byte // variable data exactly as stored (incl. any inner length word)
}

// File is a parsed calculator variable file (single variable or group).
type File struct {
	Model     Model
	Comment   string
	Variables []Variable
	// Warnings collects recoverable problems found in lenient mode
	// (bad checksum, length mismatch, trailing garbage, ...).
	Warnings []error
}

// ParseOptions control parsing.
type ParseOptions struct {
	// Strict turns every warning into an error.
	Strict bool
}

// Calculator is implemented by every supported calculator model.
type Calculator interface {
	Model() Model
	// Detect reports whether raw looks like a file of this model
	// (normally by checking the signature).
	Detect(raw []byte) bool
	Parse(raw []byte, opt ParseOptions) (*File, error)
	// Marshal builds a complete file (header, entries, checksum).
	Marshal(f *File) ([]byte, error)
	TypeInfo(t VarType) TypeInfo
	// Codec returns the program tokenizer/detokenizer, or nil when the model
	// does not support it yet.
	Codec() ProgramCodec
}

// Details are model-specific facts about one variable.
type Details struct {
	Kind       string // e.g. "tokenized", "plain text", "assembly"
	EditLocked bool
	// Binary is set when the text form would only be a hex dump
	// (e.g. compiled assembly programs).
	Binary bool
}

// Describer is optionally implemented by a Calculator to give details about
// a variable beyond its type.
type Describer interface {
	Details(v *Variable) Details
}

// DetokenizeOptions control conversion of tokenized data to text.
type DetokenizeOptions struct {
	// Strict makes unknown tokens an error. Otherwise they are written as
	// escapes (⟦NN⟧ / ⟦NNNN⟧) that Tokenize understands.
	Strict bool
}

// TokenizeOptions control conversion of text to tokenized data.
type TokenizeOptions struct {
	// EditLocked marks the created program as edit-locked.
	EditLocked bool
}

// ProgramCodec converts tokenized program/string/equation data to text and back.
// Output text is Unicode. Tokenize also accepts ASCII spellings of symbols
// (e.g. "->" for "→", "!=" for "≠").
type ProgramCodec interface {
	// Text renders a textual variable (program, string, equation) as text.
	Text(v *Variable, opt DetokenizeOptions) (string, error)
	// NewProgram tokenizes source text into a program variable.
	NewProgram(name, source string, opt TokenizeOptions) (*Variable, error)
	// Detokenize renders a raw token stream (no length prefix).
	Detokenize(tokens []byte, opt DetokenizeOptions) (string, error)
	// Tokenize converts text to a raw token stream (no length prefix).
	Tokenize(source string) ([]byte, error)
}

// Errors shared by implementations.
var (
	ErrUnknownFormat = errors.New("ticalc: unknown file format")
	ErrNotText       = errors.New("ticalc: variable type has no text form")
	ErrNoCodec       = errors.New("ticalc: calculator has no program codec")
)

var (
	regMu sync.RWMutex
	reg   = map[Model]Calculator{}
)

// Register makes a calculator available to Detect, Lookup and Parse.
// It is meant to be called from the init function of the implementing package.
func Register(c Calculator) {
	regMu.Lock()
	defer regMu.Unlock()
	if _, dup := reg[c.Model()]; dup {
		panic(fmt.Sprintf("ticalc: calculator %s registered twice", c.Model()))
	}
	reg[c.Model()] = c
}

// Lookup returns the registered calculator for a model.
func Lookup(m Model) (Calculator, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	c, ok := reg[m]
	return c, ok
}

// Models lists registered models, sorted.
func Models() []Model {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]Model, 0, len(reg))
	for m := range reg {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Detect returns the calculator whose Detect accepts raw.
func Detect(raw []byte) (Calculator, error) {
	for _, m := range Models() {
		c, _ := Lookup(m)
		if c.Detect(raw) {
			return c, nil
		}
	}
	if sig := Signature(raw); sig != "" {
		return nil, fmt.Errorf("%w: signature %q (calculator not supported)", ErrUnknownFormat, sig)
	}
	return nil, ErrUnknownFormat
}

// Parse auto-detects the calculator model and parses raw.
func Parse(raw []byte, opt ParseOptions) (*File, Calculator, error) {
	c, err := Detect(raw)
	if err != nil {
		return nil, nil, err
	}
	f, err := c.Parse(raw, opt)
	return f, c, err
}

// Signature returns the "**TIxx**" style signature at the start of raw, or ""
// when raw does not start with one.
func Signature(raw []byte) string {
	if len(raw) < 8 || raw[0] != '*' || raw[1] != '*' || raw[6] != '*' || raw[7] != '*' {
		return ""
	}
	return string(raw[:8])
}
