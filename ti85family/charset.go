package ti85family

import "github.com/hjkoskel/hkoTiUtil/ticalc"

// fontMap is the TI-85/TI-86 font outside printable ASCII (both models use
// the same font). The mapping was
// checked against example programs (accented words like "Sábado", "(r∠θ)",
// "tan⁻¹", "If R≠0", "√L", "Σx") and the CharCat character catalog program.
// Bytes not listed here are written as ⟦NN⟧.
// TODO: identify the remaining glyphs 0xD1..0xFF, 0x00 and 0x7F.
var fontMap = map[byte]string{
	0x01: "ᵇ", // binary number suffix
	0x02: "ᵒ", // octal number suffix
	0x03: "ᵈ", // decimal number suffix
	0x04: "ʰ", // hexadecimal number suffix
	0x05: "▶",
	0x06: "⬆", // bold arrows (CharCat: BAeroUp/BAeroDn)
	0x07: "⬇",
	0x08: "∫",
	0x09: "×",
	// Hex digits A..F have their own glyphs; fullwidth letters keep them
	// distinct from variable names.
	0x0A: "Ａ", 0x0B: "Ｂ", 0x0C: "Ｃ", 0x0D: "Ｄ", 0x0E: "Ｅ", 0x0F: "Ｆ",
	0x10: "√",
	0x11: "⁻¹",
	0x12: "²",
	0x13: "∠",
	0x14: "°",
	0x15: "ʳ",
	0x16: "ᵀ",
	0x17: "≤",
	0x18: "≠",
	0x19: "≥",
	0x1A: "⁻", // negation
	0x1B: "ᴇ", // exponent
	0x1C: "→",
	0x1D: "⏨", // small "10" of 10^ (₁₀ would clash with bytes 81 80)
	0x1E: "↑", // CharCat: AeroUp/AeroDn
	0x1F: "↓",

	0x80: "₀", 0x81: "₁", 0x82: "₂", 0x83: "₃", 0x84: "₄",
	0x85: "₅", 0x86: "₆", 0x87: "₇", 0x88: "₈", 0x89: "₉",

	0x8A: "Á", 0x8B: "À", 0x8C: "Â", 0x8D: "Ä", 0x8E: "á", 0x8F: "à", 0x90: "â", 0x91: "ä",
	0x92: "É", 0x93: "È", 0x94: "Ê", 0x95: "Ë", 0x96: "é", 0x97: "è", 0x98: "ê", 0x99: "ë",
	0x9A: "Í", 0x9B: "Ì", 0x9C: "Î", 0x9D: "Ï", 0x9E: "í", 0x9F: "ì", 0xA0: "î", 0xA1: "ï",
	0xA2: "Ó", 0xA3: "Ò", 0xA4: "Ô", 0xA5: "Ö", 0xA6: "ó", 0xA7: "ò", 0xA8: "ô", 0xA9: "ö",
	0xAA: "Ú", 0xAB: "Ù", 0xAC: "Û", 0xAD: "Ü", 0xAE: "ú", 0xAF: "ù", 0xB0: "û", 0xB1: "ü",
	0xB2: "Ç", 0xB3: "ç", 0xB4: "Ñ", 0xB5: "ñ",
	0xB6: "´", 0xB7: "ˋ", 0xB8: "¨", 0xB9: "¿", 0xBA: "¡",
	0xBB: "α", 0xBC: "β", 0xBD: "γ", 0xBE: "Δ", 0xBF: "δ", 0xC0: "ε", 0xC1: "θ",
	0xC2: "λ", 0xC3: "μ", 0xC4: "π", 0xC5: "ρ", 0xC6: "Σ", 0xC7: "σ", 0xC8: "τ",
	0xC9: "φ", 0xCA: "Ω", 0xCB: "x̄", 0xCC: "ȳ", 0xCD: "ˣ", 0xCE: "…",
	0xCF: "◀", // CharCat: TAeroLt (0x05 ▶ is TAeroRt)
	0xD0: "■", // CharCat: BBox
}

// Charset is the font mapping used for names, strings, labels and comments.
var Charset = ticalc.NewCharset(true, fontMap)

// TextCharset is Charset plus the "hard return" (0xD6) as a newline. It is
// used for plain-text (untokenized) programs and AsmPrgm hex listings.
var TextCharset = func() *ticalc.Charset {
	m := map[byte]string{0xD6: "\n"}
	for k, v := range fontMap {
		m[k] = v
	}
	return ticalc.NewCharset(true, m)
}()
