package ticalc

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Container describes the "**TIxx**" file layout shared by the TI-85 and
// TI-86 (and, with small differences, other models):
//
//	00  8  signature, e.g. "**TI86**"
//	08  3  further signature: 1A 0A 00 (TI-86), 1A 0C 00 (TI-85)
//	0B 42  comment, NUL/space padded
//	35  2  data length n
//	37  n  variable entries
//	37+n 2 checksum = sum of data bytes & 0xFFFF
//
// Each entry:
//
//	00 2 header length h, counted from offset 02 (TI-86: 12, TI-85: 4+name length)
//	02 2 data length
//	04 1 type
//	05 1 name length
//	06 . name (h-4 bytes; padded to 8 on the TI-86)
//	2+h 2 data length again
//	4+h . data
type Container struct {
	Model     Model
	Signature string
	// Magic is the further signature after Signature. Zero means 1A 0A 00.
	Magic [3]byte
	// PadNames writes names padded to 8 bytes (TI-86 style). When false, the
	// header is 4+len(name) bytes (TI-85 style). Parsing accepts both.
	PadNames bool
	Charset  *Charset
}

const (
	headerLen  = 0x37
	commentLen = 42
)

func (c *Container) magic() [3]byte {
	if c.Magic == [3]byte{} {
		return [3]byte{0x1A, 0x0A, 0x00}
	}
	return c.Magic
}

// Detect reports whether raw starts with this container's signature.
func (c *Container) Detect(raw []byte) bool {
	return len(raw) >= 11 && string(raw[:8]) == c.Signature
}

// Parse decodes a container file. In lenient mode recoverable problems are
// collected in File.Warnings and as much data as possible is returned.
func (c *Container) Parse(raw []byte, opt ParseOptions) (*File, error) {
	f := &File{Model: c.Model}
	warn := func(format string, a ...any) error {
		err := fmt.Errorf(format, a...)
		if opt.Strict {
			return err
		}
		f.Warnings = append(f.Warnings, err)
		return nil
	}
	if !c.Detect(raw) {
		return nil, fmt.Errorf("%w: missing %q signature", ErrUnknownFormat, c.Signature)
	}
	if len(raw) < headerLen {
		return nil, fmt.Errorf("file too short (%d bytes)", len(raw))
	}
	if [3]byte(raw[8:11]) != c.magic() {
		if err := warn("unexpected bytes after signature: % X", raw[8:11]); err != nil {
			return nil, err
		}
	}
	comment := raw[11 : 11+commentLen]
	if i := indexByte(comment, 0); i >= 0 {
		comment = comment[:i]
	}
	f.Comment = strings.TrimRight(c.Charset.Decode(comment), " ")

	dataLen := int(binary.LittleEndian.Uint16(raw[0x35:0x37]))
	avail := len(raw) - headerLen
	switch {
	case avail < dataLen+2:
		if err := warn("data length %d but only %d bytes in file", dataLen, avail); err != nil {
			return nil, err
		}
		if dataLen > avail {
			dataLen = avail
		}
	case avail > dataLen+2:
		if err := warn("%d extra bytes after checksum", avail-dataLen-2); err != nil {
			return nil, err
		}
	}
	data := raw[headerLen : headerLen+dataLen]
	if headerLen+dataLen+2 <= len(raw) {
		stored := binary.LittleEndian.Uint16(raw[headerLen+dataLen:])
		if sum := Checksum(data); sum != stored {
			if err := warn("checksum mismatch: stored %04X, computed %04X", stored, sum); err != nil {
				return nil, err
			}
		}
	}

	for i := 0; i < len(data); {
		if len(data)-i < 6 {
			if err := warn("%d stray bytes at end of data", len(data)-i); err != nil {
				return nil, err
			}
			break
		}
		h := int(binary.LittleEndian.Uint16(data[i:]))
		vlen := int(binary.LittleEndian.Uint16(data[i+2:]))
		if h < 4 || h > 12 {
			return f, fmt.Errorf("entry at %d: bad header length %d", i, h)
		}
		if i+4+h > len(data) {
			return f, fmt.Errorf("entry at %d: truncated header", i)
		}
		v := Variable{Type: VarType(data[i+4])}
		nameLen := int(data[i+5])
		if nameLen > h-4 {
			if err := warn("entry at %d: name length %d exceeds header", i, nameLen); err != nil {
				return f, err
			}
			nameLen = h - 4
		}
		v.RawName = append([]byte(nil), data[i+6:i+6+nameLen]...)
		v.Name = c.Charset.Decode(v.RawName)
		if dup := int(binary.LittleEndian.Uint16(data[i+2+h:])); dup != vlen {
			if err := warn("variable %q: length %d vs repeated %d", v.Name, vlen, dup); err != nil {
				return f, err
			}
		}
		start := i + 4 + h
		end := start + vlen
		if end > len(data) {
			if err := warn("variable %q truncated: need %d bytes, have %d", v.Name, vlen, len(data)-start); err != nil {
				return f, err
			}
			end = len(data)
		}
		v.Data = append([]byte(nil), data[start:end]...)
		f.Variables = append(f.Variables, v)
		i = end
	}
	return f, nil
}

// Marshal writes a container file.
func (c *Container) Marshal(f *File) ([]byte, error) {
	var data []byte
	for _, v := range f.Variables {
		name := v.RawName
		if name == nil {
			var err error
			if name, err = c.Charset.Encode(v.Name); err != nil {
				return nil, fmt.Errorf("variable %q: %w", v.Name, err)
			}
		}
		if len(name) == 0 || len(name) > 8 {
			return nil, fmt.Errorf("variable %q: name must be 1..8 bytes", v.Name)
		}
		if len(v.Data) > 0xFFFF {
			return nil, fmt.Errorf("variable %q: data too large", v.Name)
		}
		h := 4 + len(name)
		if c.PadNames {
			h = 12
		}
		e := make([]byte, 4+h)
		binary.LittleEndian.PutUint16(e[0:], uint16(h))
		binary.LittleEndian.PutUint16(e[2:], uint16(len(v.Data)))
		e[4] = byte(v.Type)
		e[5] = byte(len(name))
		for k := 6; k < 2+h; k++ {
			e[k] = ' '
		}
		copy(e[6:], name)
		binary.LittleEndian.PutUint16(e[2+h:], uint16(len(v.Data)))
		data = append(data, e...)
		data = append(data, v.Data...)
	}
	if len(data) > 0xFFFF {
		return nil, fmt.Errorf("file data too large (%d bytes); split into several files", len(data))
	}
	comment, err := c.Charset.Encode(f.Comment)
	if err != nil {
		return nil, fmt.Errorf("comment: %w", err)
	}
	if len(comment) > commentLen {
		comment = comment[:commentLen]
	}
	out := make([]byte, headerLen, headerLen+len(data)+2)
	copy(out, c.Signature)
	m := c.magic()
	copy(out[8:], m[:])
	copy(out[11:], comment)
	binary.LittleEndian.PutUint16(out[0x35:], uint16(len(data)))
	out = append(out, data...)
	out = binary.LittleEndian.AppendUint16(out, Checksum(data))
	return out, nil
}

// Checksum is the 16-bit sum of all bytes.
func Checksum(b []byte) uint16 {
	var s uint16
	for _, x := range b {
		s += uint16(x)
	}
	return s
}

func indexByte(b []byte, x byte) int {
	for i, v := range b {
		if v == x {
			return i
		}
	}
	return -1
}
