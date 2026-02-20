package ansi

// Token represents a lexical token in an ANSI stream.
type Token struct {
	Type TokenType
	Text string // The raw text of the token
	// For CSIToken: contains the parameter bytes without the ESC [ prefix
	// For TextToken: contains the literal text
}

// TokenType identifies the kind of token.
type TokenType int

const (
	TextToken TokenType = iota
	CSIToken
	ESCToken
	EOFToken
)

// Lexer tokenizes a string containing ANSI escape sequences.
type Lexer struct {
	input string
	pos   int
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

// Next returns the next token from the input.
func (l *Lexer) Next() Token {
	if l.pos >= len(l.input) {
		return Token{Type: EOFToken}
	}

	ch := l.input[l.pos]

	if ch == '\x1b' {
		return l.readEscape()
	}

	return l.readText()
}

// readText reads a run of non-escape characters.
func (l *Lexer) readText() Token {
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '\x1b' {
		l.pos++
	}
	return Token{
		Type: TextToken,
		Text: l.input[start:l.pos],
	}
}

// readEscape reads an escape sequence starting with ESC.
func (l *Lexer) readEscape() Token {
	start := l.pos
	l.pos++ // consume ESC

	if l.pos >= len(l.input) {
		return Token{
			Type: TextToken,
			Text: l.input[start:l.pos],
		}
	}

	if l.input[l.pos] == '[' {
		return l.readCSI(start)
	}

	// Other escape sequence (ESC followed by single char)
	l.pos++
	return Token{
		Type: ESCToken,
		Text: l.input[start:l.pos],
	}
}

// readCSI reads a CSI sequence starting after ESC [.
func (l *Lexer) readCSI(start int) Token {
	l.pos++ // consume '['

	// Read parameter bytes (0x30-0x3F) and intermediate bytes (0x20-0x2F)
	for l.pos < len(l.input) {
		b := l.input[l.pos]
		// Parameter bytes: 0x30-0x3F (includes digits, semicolon, etc.)
		// Intermediate bytes: 0x20-0x2F
		if (b >= 0x30 && b <= 0x3F) || (b >= 0x20 && b <= 0x2F) {
			l.pos++
		} else {
			break
		}
	}

	// Read final byte (0x40-0x7E)
	if l.pos < len(l.input) {
		b := l.input[l.pos]
		if b >= 0x40 && b <= 0x7E {
			l.pos++
		}
	}

	return Token{
		Type: CSIToken,
		Text: l.input[start:l.pos],
	}
}

// ParseCSI extracts parameters from a CSI token.
// Returns the parameter bytes (between ESC [ and final byte) and the final byte.
func ParseCSI(token Token) (params string, final byte) {
	if token.Type != CSIToken || len(token.Text) < 3 {
		return "", 0
	}

	// Skip ESC [
	text := token.Text[2:]
	if len(text) == 0 {
		return "", 0
	}

	// Last byte is the final byte
	final = text[len(text)-1]

	// Everything before final byte is parameters
	if len(text) > 1 {
		params = text[:len(text)-1]
	}

	return params, final
}

// ParseSGRParams parses SGR (Select Graphic Rendition) parameters from a string.
// Returns a slice of integer parameters.
func ParseSGRParams(s string) []int {
	if s == "" {
		return []int{0}
	}

	var params []int
	var current int
	hasCurrent := false

	for _, b := range []byte(s) {
		if b >= '0' && b <= '9' {
			current = current*10 + int(b-'0')
			hasCurrent = true
		} else if b == ';' || b == ':' {
			if hasCurrent {
				params = append(params, current)
			} else {
				params = append(params, 0)
			}
			current = 0
			hasCurrent = false
		}
		// Ignore other bytes (intermediate bytes, etc.)
	}

	if hasCurrent {
		params = append(params, current)
	} else if len(params) == 0 {
		params = append(params, 0)
	}

	return params
}
