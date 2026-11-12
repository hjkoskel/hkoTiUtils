# HKOTIUTILPLAN: hkoTiUtil as a Go library + CLI tools

Goal: a Go library (`github.com/hjkoskel/hkoTiUtil`) that reads TI calculator
variable files (TI-86 `.86?` and TI-85 `.85?` files), including files inside
`.zip` archives and zips nested in zips. It prints tokenized TI-BASIC programs
as clean Unicode text and tokenizes text back into programs. The design
supports adding more calculator models (TI-85, TI-83, …) later.

Status 2026-11-12: phases 0–4 are done for the TI-86 and TI-85. Pictures
decode (image + Unicode block drawing) and the Fyne GUI `tibrowser` exists.
The rest of phase 5 (other variable types) remains open.

---

## 1. Decisions

| Topic | Decision |
|---|---|
| Module path | `github.com/hjkoskel/hkoTiUtil` |
| Text output | Unicode (`→ ≠ ≤ ≥ ▶ √ π θ ⁻ ²`) |
| Tokenizer input | Unicode **and** ASCII digraphs: `->`, `!=`, `=/=`, `<=`, `>=`, `>Dec`…, `/_`, `~` (negation), `li>vc`, `Eq>St` |
| Line prefix | None (no `:` at start of lines) |
| `download/` | Kept out of git (`.gitignore`). Only `download/basic/ti86/*.txt` URL lists are tracked |
| Other calculators | Common `ticalc` package with interfaces + registry. TI-86 and TI-85 implemented |
| TI-85/86 token engine | Shared package `ti85family`; each model has a `Dialect` (token table + type codes) |
| Old code | Moved to `legacy/` (`//go:build ignore`), kept for reference. Can be deleted |

---

## 2. Layout

```
hkoTiUtil/
├── go.mod                    module github.com/hjkoskel/hkoTiUtil (go 1.22; Fyne is used only by cmd/tibrowser)
├── docs/ti86prot.txt         Tim Singer's TI-86 format guide
├── ticalc/                   calculator-independent core
│   ├── ticalc.go             Calculator / ProgramCodec / Describer interfaces, File, Variable, registry
│   ├── image.go              Imager interface, ImageText (Unicode block drawing)
│   ├── container.go          shared "**TIxx**" file container (parse + marshal, lenient/strict)
│   └── charset.go            byte<->Unicode charset with ⟦NN⟧ escapes
├── ti85family/               token engine shared by TI-85 and TI-86
│   ├── dialect.go            Dialect (token table + type codes), implements ProgramCodec
│   ├── tokens.go             StandardTokens table (Unicode text, ASCII alternatives, spacing flags)
│   ├── charset.go            TI-85/86 font → Unicode
│   ├── decode.go / encode.go bytes <-> []Token
│   ├── detokenize.go         tokens → text (self-checking, exact round trip)
│   ├── tokenize.go           text → tokens (lexer)
│   ├── picture.go            picture bitmap (16 bytes/row) → image.Paletted
│   └── program.go            program layout: Inspect (tokenized/plain/asm, edit lock), Text, NewProgram
├── ti86/                     TI-86: calculator, type codes, Dialect + convenience wrappers
├── ti85/                     TI-85: calculator, type codes, Dialect
├── tiarchive/                walk dirs, files, zips, nested zips (size/depth limits)
├── internal/cli/             shared helpers for the tools
├── cmd/
│   ├── tilist/               list variables (table or -json)
│   ├── tibasic/              print programs as text (-o dir, -all, -asm, -strict)
│   ├── tiextract/            one standalone file per variable + .txt
│   ├── titokenize/           text → .86p (-name, -locked, -o)
│   ├── tistats/              corpus statistics, remaining escapes
│   └── tibrowser/            Fyne GUI: variable list + preview, copy/paste, save as text/PNG
├── corpus_test.go            corpus round-trip tests for every registered model
├── testdata/                 small samples (.86?, .85?) + golden/*.txt
└── legacy/                   old code, not built
```

---

## 3. Multi-calculator architecture

```go
package ticalc

type Calculator interface {
    Model() Model                                  // "TI-86"
    Detect(raw []byte) bool                        // signature check
    Parse(raw []byte, opt ParseOptions) (*File, error)
    Marshal(f *File) ([]byte, error)
    TypeInfo(t VarType) TypeInfo                   // name, extension, Text/Program flags
    Codec() ProgramCodec                           // nil if programs are not supported yet
}

type ProgramCodec interface {
    Text(v *Variable, opt DetokenizeOptions) (string, error)          // program/string/equation
    NewProgram(name, src string, opt TokenizeOptions) (*Variable, error)
    Detokenize(tokens []byte, opt DetokenizeOptions) (string, error)
    Tokenize(src string) ([]byte, error)
}

type Describer interface { Details(v *Variable) Details }  // optional: kind, edit lock, binary

func Register(c Calculator)                // called from the model package's init()
func Detect(raw []byte) (Calculator, error)
func Parse(raw []byte, opt ParseOptions) (*File, Calculator, error)
```

`File`, `Variable`, `VarType`, `ParseOptions` and the errors are shared.
`ticalc.Container` implements the `**TIxx**` layout. It accepts both padded
(TI-86) and unpadded (TI-85) entry headers, and is configured with
signature, name padding and charset.

The tools only use `ticalc` and import `ti85`/`ti86` for their side effect (in
`internal/cli`). A new model works in every tool once it is imported there.

The TI-85/TI-86 token engine lives in `ti85family`. A model creates a
`Dialect` with `ti85family.NewDialect(Config{Model, Tokens, Types, TypeName})`;
the dialect implements `ticalc.ProgramCodec` and the `Describer` logic.

### Adding a calculator

The TI-85 (`ti85/ti85.go`, one file) is the template:

1. A `ticalc.Container` with the model's signature, `Magic` (the 3 bytes after
   the signature) and name padding.
2. The type code table (`TypeInfo`: name, extension, Text/Program flags).
3. A codec: for the TI-85/86 family, `ti85family.NewDialect` with a token
   table. For other families, `Codec()` can return `nil` at first, and
   listing and extraction still work.
4. `func init() { ticalc.Register(Calc) }`, and a blank import in `internal/cli/cli.go`.
5. Testdata + golden test. `corpus_test.go` covers every registered model automatically.

The TI-83/82/73 and TI-89/92 use a different header (`**TI83F*`, `**TI89**`
with other entry layouts) and an unrelated token scheme, so they need their own
container options and token engine. The `ticalc` interfaces stay the same.

---

## 4. Format findings

### TI-86 (from the 2,381-file corpus)

1. **`Goto` (0xE1) = 2 cache bytes + zero-terminated label.** The old code read the
   cache as the label, which produced `[0x00]Menu P4Reg ==` garbage. The cache is a
   runtime jump address (0 in 12,801 Gotos, arbitrary values in 7,156), so the
   tokenizer writes `00 00`.
2. **The entry header length varies.** The first word is `4 + name length` when names are
   unpadded (TI-85 style; 305 entries). Bytes after the name length can be garbage
   (`A.86n` stores "A" followed by "UTOPELI").
3. **Program kinds:** tokenized 1,635, plain text (first byte 00) 779, plain text
   edit-locked (00 00) 452, assembly (`8E 28`) 367, tokenized edit-locked (`8E 29`) 196.
   Plain-text programs use `0xD6` as the line break.
4. **Tokens with a payload that the reference doc does not describe:**
   - `8E 32` = 2 bytes + length + name (a call to an external routine, e.g. `EXISTS`, `LnOff`)
   - `FF` = zero-terminated text, used as a comment hack
5. **Number literals (0x44)** use the TI font: `0x1B` = ᴇ, `0x1A` = ⁻, `0x0A–0x0F`
   = hex digits A–F, `0x01–0x04` = base suffixes b o d h.
6. **Strings (type 0x0C) are charset text, not tokens.** Many hold binary sprite data
   for assembly games, which is why ⟦NN⟧ escapes appear there.
7. **Name tokens:** `0x32` x y t r θ Q1–Q9 exp eqn. `0x3B` y1… xt1… yt1… r1….
   `0x3C` Ans, statistics vars. `0x33–0x3A` other names of 1–8 bytes.
8. **Damaged files:** 17 have a bad checksum, 12 have trailing bytes, and a few are
   truncated. Lenient mode warns, strict mode fails. Other formats: 13 `********`,
   12 `**TI85**`, 3 `**TI83**`, 3 `**TI89**`, 2 `**TI82**`. 3 zip entries use
   Implode (method 6), which Go cannot read; these are reported as warnings.

### TI-85 (21 games from ticalc.org/pub/85/basic/games + 12 files above)

1. The signature is `**TI85**` followed by `1A 0C 00` (the TI-86 has `1A 0A 00`).
2. Entry headers are unpadded: header length = 4 + name length. A few files
   pad anyway, and the parser accepts both.
3. Same program layout, font, plain-text form (`0xD6` newline) and `Goto` cache
   as the TI-86. All 68 tokenized TI-85 programs decode with the shared table and
   round-trip exactly. They use only one-byte and `3D xx` tokens, never `8E xx`.
4. Some games ship as both `.85g` and `.86g`. Both decode to the same text.
5. The type codes used in the samples (Real, Matrix, String, Picture, Program)
   match the TI-86 codes.

---

## 5. Text format

- **Exact:** for every program, `Tokenize(Detokenize(x)) == x`, except that `Goto`
  caches are zeroed. Verified on all tokenized programs and equations in
  the corpus (2,227 TI-86 + 68 TI-85).
- **Spacing:** the renderer asks the lexer whether two tokens would merge, looking a
  few tokens ahead (`6 E-7` must not become the number `6ᴇ⁻7`). Statement words
  get a space after them (`If X`, `Disp "A"`), infix words get spaces around
  them (`A or B`). Otherwise nothing is added: `Circl(5,5,3)`, `2X`, `sin X`.
- **Escapes:** `⟦NN⟧` is an unknown 1-byte token or glyph, and `⟦NNNN⟧` is a 2-byte
  token. Payload tokens are written as `⟦FF⟧"text"` and `⟦8E32⟧⟦0400⟧"EXISTS"`.
  A truncated tail is written as `⟦#hex⟧`. Compiled assembly is `⟦8E28⟧` plus hex
  lines (`tibasic` skips it unless `-asm` is given).
- **Symbols that differ from plain ASCII:** `₁₀^` (10^ token), `ℯ^` and `ℯ`
  (constant), `⁻` (negation; ASCII `~`), `ᴇ` (exponent; ASCII `E` before a
  digit is also accepted), `Ａ–Ｆ` (hex digits), `ᵇ ᵒ ᵈ ʰ` (base suffixes), `⏨`
  (charset 0x1D, small "10").
- **Name tokens** are chosen from the name (`x` → 0x32, `y1` → 0x3B, `Ans` → 0x3C).
  If the original used a different token, the escape form is used, so nothing is lost.

---

## 6. Open work

### 6.1 TI-86
- [ ] Charset: identify glyphs `0xD1–0xFF`, `0x00`, `0x7F` (verify in an emulator or
      with a font dump). They are written as `⟦NN⟧` until then.
- [ ] Meaning of `8E 32` bytes 2–3 and better text for it (now an escape).
- [x] Picture (0x11): 1008 bytes = 128×63, 16 bytes per row, MSB leftmost
      (1024-byte 128×64 pictures also occur). `ticalc.Imager` on both models.
- [ ] Phase 5: Real/Complex (10-byte BCD), List, Vector, Matrix → text
      (`tibrowser` shows a hex dump until then); Picture → PNG in `tiextract`;
      GDB/window dumps.
- [ ] Plain-text program creation (`TokenizeOptions.PlainText`) if needed.
- [ ] Backup files (type 0x1D, multi-part).

### 6.2 TI-85
- [x] `ti85` package: container, types, dialect, registered in the tools.
- [x] Token engine moved to `ti85family` and shared with the TI-86.
- [ ] Token table: TI-85 currently uses the full TI-85/86 table. Check against a
      TI-85 token reference, drop TI-86-only tokens (probably all `8E xx`), and
      rename any differing spellings.
- [ ] Verify the TI-85 type codes 0x17–0x1B and the `.85r` extension for range settings
      (only Real, Matrix, String, Picture and Program appear in the samples).
- [ ] TI-85 backup files.

### 6.3 Other calculators
- [ ] TI-82/83/73 (different header layout and token engine).
- [ ] TI-89/92 (different container, and programs are tokenized differently).

### 6.4 Tooling
- [ ] Implode support for old zips (3 files): small `explode` decompressor via
      `zip.RegisterDecompressor`.
- [ ] README examples, CI (`go vet`, `go test`, short fuzz).
- [x] `tibrowser` (Fyne): file/group/zip/dir argument or Open dialog; split view
      with a variable table + preview when there is more than one variable (columns
      Name with a per-type icon, Type, Size, and Source when there are several files;
      click a header to sort, again to reverse); editable preview;
      copy (selection or all), paste, save as text, save picture as PNG.
- [ ] `tibrowser`: the Fyne clipboard is text-only, so pictures are copied as
      block drawings. Real image copy would need a platform clipboard library.
- [x] `tibrowser`: file dialogs sized to 90% of the window (Fyne dialogs are drawn
      inside the window and cannot be resized by the user). `-tags flatpak` uses
      the native dialog via xdg-desktop-portal on Linux (builds; not tried by hand).
- [ ] `tibrowser`: save edited program text back as a calculator file (tokenize).

---

## 7. Testing

| Test | What |
|---|---|
| `ti86` `TestGolden` | `testdata/*.86p` → `testdata/golden/*.txt` (`go test ./ti86 -update` rewrites) |
| `ti86` `TestTokenize*` | byte-exact token cases, ASCII vs Unicode equivalence, errors |
| `ti86` `TestNewProgramFile` | text → program → file → parse → identical text |
| `ti86` `TestCharsetPairs` | all 65,536 byte pairs survive the charset round trip |
| `ti85` | golden (`*.85p`), new-file header/unpadded names, model detection |
| root `TestCorpus*` | whole `download/` corpus for every registered model: program round trip, re-marshal of every file, strings and names (skipped when `download/` is missing) |
| `ticalc` | container round trip (padded/unpadded), lenient vs strict, truncation, `FuzzContainerParse` |
| `tiarchive` | nested zips, depth limit, callback error stops the walk |
| `ti86` `TestPicture` | `testdata/TT3DPIC.86i` → block drawing in `testdata/golden/TT3DPic.pic.txt` |
| `cmd/tibrowser` | headless (Fyne test driver): split view for groups, single file, picture + copy/paste, hex dump |

Run: `go test ./...`, and `go test ./ticalc -fuzz FuzzContainerParse -fuzztime 30s` for fuzzing.
