package ti85family

import (
	"encoding/binary"
	"fmt"

	"github.com/hjkoskel/hkoTiUtil/ticalc"
)

// ProgramKind tells how a program variable is stored.
type ProgramKind int

const (
	Tokenized ProgramKind = iota // TI-BASIC tokens
	PlainText                    // untokenized text (first byte 00)
	Assembly                     // compiled Z80 code (starts 8E 28)
)

func (k ProgramKind) String() string {
	return [...]string{"tokenized", "plain text", "assembly"}[k]
}

// ProgramInfo describes a program variable.
type ProgramInfo struct {
	Kind       ProgramKind
	EditLocked bool
	// Body is the token stream (Tokenized/Assembly, without the edit-lock
	// marker) or the text bytes (PlainText).
	Body []byte
}

// Inspect classifies a program variable (docs/ti86prot.txt §3.11):
// 00 → plain text, 00 00 → edit-locked plain text, 8E 29 → edit-locked
// tokens, 8E 28 → compiled assembly, anything else → tokens.
func (dl *Dialect) Inspect(v *ticalc.Variable) (ProgramInfo, error) {
	if v.Type != dl.types.Program {
		return ProgramInfo{}, fmt.Errorf("%q is a %s, not a program", v.Name, dl.typeName(v.Type))
	}
	body := InnerData(v.Data)
	var pi ProgramInfo
	switch {
	case len(body) >= 1 && body[0] == 0:
		pi.Kind = PlainText
		pi.EditLocked = len(body) >= 2 && body[1] == 0
		if pi.EditLocked {
			pi.Body = body[2:]
		} else {
			pi.Body = body[1:]
		}
	case len(body) >= 2 && body[0] == 0x8E && body[1] == 0x29:
		pi.EditLocked = true
		pi.Body = body[2:]
	case len(body) >= 2 && body[0] == 0x8E && body[1] == 0x28:
		pi.Kind = Assembly
		pi.Body = body
	default:
		pi.Body = body
	}
	return pi, nil
}

// Details implements the ticalc.Describer logic for this family.
func (dl *Dialect) Details(v *ticalc.Variable) ticalc.Details {
	pi, err := dl.Inspect(v)
	if err != nil {
		return ticalc.Details{}
	}
	return ticalc.Details{Kind: pi.Kind.String(), EditLocked: pi.EditLocked, Binary: pi.Kind == Assembly}
}

// InnerData returns the bytes after a 2-byte length word, clipped to it.
func InnerData(d []byte) []byte {
	if len(d) < 2 {
		return nil
	}
	n := int(binary.LittleEndian.Uint16(d))
	if n > len(d)-2 {
		n = len(d) - 2
	}
	return d[2 : 2+n]
}

// Text renders a program, string or equation. The edit-lock marker of a
// program is not part of the text (see Inspect).
func (dl *Dialect) Text(v *ticalc.Variable, opt ticalc.DetokenizeOptions) (string, error) {
	switch v.Type {
	case dl.types.Program:
		pi, err := dl.Inspect(v)
		if err != nil {
			return "", err
		}
		if pi.Kind == PlainText {
			return dl.textCS.Decode(pi.Body), nil
		}
		return dl.Detokenize(pi.Body, opt)
	case dl.types.Equation:
		return dl.Detokenize(InnerData(v.Data), opt)
	case dl.types.String:
		return dl.cs.Decode(InnerData(v.Data)), nil
	}
	return "", fmt.Errorf("%w: %s", ticalc.ErrNotText, dl.typeName(v.Type))
}

// NewProgram tokenizes source into a program variable.
func (dl *Dialect) NewProgram(name, src string, opt ticalc.TokenizeOptions) (*ticalc.Variable, error) {
	raw, err := dl.cs.Encode(name)
	if err != nil {
		return nil, fmt.Errorf("program name: %w", err)
	}
	if len(raw) < 1 || len(raw) > 8 {
		return nil, fmt.Errorf("program name %q must be 1..8 characters", name)
	}
	body, err := dl.Tokenize(src)
	if err != nil {
		return nil, err
	}
	if opt.EditLocked {
		body = append([]byte{0x8E, 0x29}, body...)
	}
	if len(body) > 0xFFFF-2 {
		return nil, fmt.Errorf("program too large (%d bytes)", len(body))
	}
	data := binary.LittleEndian.AppendUint16(nil, uint16(len(body)))
	return &ticalc.Variable{Type: dl.types.Program, Name: name, RawName: raw, Data: append(data, body...)}, nil
}
