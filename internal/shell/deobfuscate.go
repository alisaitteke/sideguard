package shell

import (
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// maxDecodeDepth caps recursive decode rounds (nested encodings).
	maxDecodeDepth = 3
	// maxInputBytes caps the input size the decoder will scan (DoS guard).
	maxInputBytes = 64 * 1024
	// decodeBudget is the total wall-clock time the decoder may spend.
	decodeBudget = 20 * time.Millisecond
	// minTokenLen is the shortest encoded token worth attempting to decode.
	minTokenLen = 8
)

var (
	base64TokenRE = regexp.MustCompile(`[A-Za-z0-9+/]{8,}={0,2}`)
	hexEscapeRE   = regexp.MustCompile(`(?:\\x[0-9A-Fa-f]{2})+`)
	hexBlobRE     = regexp.MustCompile(`\b(?:[0-9A-Fa-f]{2}){8,}\b`)
	ansiCQuoteRE  = regexp.MustCompile(`\$'((?:[^'\\]|\\.)*)'`)
)

// Deobfuscate applies layered, static decoding (base64, hex, ANSI-C $'…') to
// reveal payloads hidden inside a command. Decoded content is APPENDED to the
// working string (joined with "; ") so re-parsing surfaces it as additional
// commands for the detect engine. It NEVER executes, evals, or spawns anything.
//
// Guards: input capped at 64KB, at most 3 decode rounds, a 20ms wall-clock
// budget, and each round re-parsed to confirm the result is still shell-like.
// Already-decoded tokens are remembered to avoid re-decoding loops.
//
// See docs/plans/2026-07-02-0001-shell-detect-history/ (sdh-phase-1.0-shell.md).
func Deobfuscate(command string) (string, DeobfuscateMeta) {
	meta := DeobfuscateMeta{}
	if len(command) == 0 || len(command) > maxInputBytes {
		return command, meta
	}
	if !utf8.ValidString(command) {
		return command, meta
	}

	deadline := time.Now().Add(decodeBudget)
	seen := make(map[string]struct{})
	current := command
	frontier := command

	for depth := 0; depth < maxDecodeDepth; depth++ {
		if time.Now().After(deadline) {
			break
		}
		decoded, layers := decodeLayer(frontier, seen)
		if len(decoded) == 0 {
			break
		}
		meta.Depth = depth + 1
		meta.Layers = append(meta.Layers, layers...)
		frontier = strings.Join(decoded, " ; ")
		current = current + " ; " + frontier
	}

	return current, meta
}

// decodeLayer scans one text for encoded tokens and returns the newly decoded
// pieces plus the decoder names that fired. seen prevents re-decoding a token.
func decodeLayer(text string, seen map[string]struct{}) (pieces []string, layers []string) {
	// ANSI-C quoting: $'...\x41...' → literal bytes.
	for _, m := range ansiCQuoteRE.FindAllStringSubmatch(text, -1) {
		token := m[0]
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		if decoded, ok := decodeANSIC(m[1]); ok && plausibleText(decoded) {
			pieces = append(pieces, decoded)
			layers = append(layers, "ansi-c")
		}
	}

	// \xHH escape sequences.
	for _, token := range hexEscapeRE.FindAllString(text, -1) {
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		if decoded, ok := decodeHexEscapes(token); ok && plausibleText(decoded) {
			pieces = append(pieces, decoded)
			layers = append(layers, "hex")
		}
	}

	// Bare hex blobs (even-length runs of hex digits).
	for _, token := range hexBlobRE.FindAllString(text, -1) {
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		if decoded, ok := decodeHexBlob(token); ok && plausibleText(decoded) {
			pieces = append(pieces, decoded)
			layers = append(layers, "hex")
		}
	}

	// base64 tokens.
	for _, token := range base64TokenRE.FindAllString(text, -1) {
		if len(token) < minTokenLen {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		if decoded, ok := decodeBase64(token); ok && plausibleText(decoded) {
			pieces = append(pieces, decoded)
			layers = append(layers, "base64")
		}
	}

	return pieces, layers
}

func decodeBase64(token string) (string, bool) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if raw, err := enc.DecodeString(token); err == nil && len(raw) > 0 {
			return string(raw), true
		}
	}
	return "", false
}

func decodeHexEscapes(token string) (string, bool) {
	var b strings.Builder
	for i := 0; i+4 <= len(token); i += 4 {
		// token is a run of "\xHH" sequences (4 bytes each).
		v, err := strconv.ParseUint(token[i+2:i+4], 16, 8)
		if err != nil {
			return "", false
		}
		b.WriteByte(byte(v))
	}
	if b.Len() == 0 {
		return "", false
	}
	return b.String(), true
}

func decodeHexBlob(token string) (string, bool) {
	raw, err := hex.DecodeString(token)
	if err != nil || len(raw) == 0 {
		return "", false
	}
	return string(raw), true
}

// decodeANSIC interprets a subset of ANSI-C ($'…') escapes: \n \t \r \\ \' \"
// \xHH hex, and \NNN octal. Unknown escapes are passed through literally.
func decodeANSIC(body string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] != '\\' || i+1 >= len(body) {
			b.WriteByte(body[i])
			continue
		}
		i++
		switch body[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '\\':
			b.WriteByte('\\')
		case '\'':
			b.WriteByte('\'')
		case '"':
			b.WriteByte('"')
		case 'x':
			if i+2 < len(body) {
				if v, err := strconv.ParseUint(body[i+1:i+3], 16, 8); err == nil {
					b.WriteByte(byte(v))
					i += 2
					continue
				}
			}
			b.WriteByte('x')
		default:
			if body[i] >= '0' && body[i] <= '7' {
				end := i
				for end < len(body) && end < i+3 && body[end] >= '0' && body[end] <= '7' {
					end++
				}
				if v, err := strconv.ParseUint(body[i:end], 8, 16); err == nil {
					b.WriteByte(byte(v))
					i = end - 1
					continue
				}
			}
			b.WriteByte(body[i])
		}
	}
	if b.Len() == 0 {
		return "", false
	}
	return b.String(), true
}

// plausibleText rejects decoded blobs that are binary garbage: it requires
// valid UTF-8 and a high ratio of printable runes, so random base64/hex noise
// is not appended to the working string.
func plausibleText(s string) bool {
	if s == "" || !utf8.ValidString(s) {
		return false
	}
	printable := 0
	total := 0
	for _, r := range s {
		total++
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			printable++
		}
	}
	if total == 0 {
		return false
	}
	return float64(printable)/float64(total) >= 0.9
}
