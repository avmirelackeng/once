package ansi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	t.Run("PlainText", func(t *testing.T) {
		l := NewLexer("Hello World")

		tok := l.Next()
		assert.Equal(t, TextToken, tok.Type)
		assert.Equal(t, "Hello World", tok.Text)

		tok = l.Next()
		assert.Equal(t, EOFToken, tok.Type)
	})

	t.Run("TextWithCSI", func(t *testing.T) {
		l := NewLexer("Hello\x1b[31mRed\x1b[0mWorld")

		tok := l.Next()
		assert.Equal(t, TextToken, tok.Type)
		assert.Equal(t, "Hello", tok.Text)

		tok = l.Next()
		assert.Equal(t, CSIToken, tok.Type)
		assert.Equal(t, "\x1b[31m", tok.Text)

		tok = l.Next()
		assert.Equal(t, TextToken, tok.Type)
		assert.Equal(t, "Red", tok.Text)

		tok = l.Next()
		assert.Equal(t, CSIToken, tok.Type)
		assert.Equal(t, "\x1b[0m", tok.Text)

		tok = l.Next()
		assert.Equal(t, TextToken, tok.Type)
		assert.Equal(t, "World", tok.Text)

		tok = l.Next()
		assert.Equal(t, EOFToken, tok.Type)
	})

	t.Run("MultipleCSISequences", func(t *testing.T) {
		l := NewLexer("\x1b[1;31m\x1b[44mText")

		tok := l.Next()
		assert.Equal(t, CSIToken, tok.Type)
		assert.Equal(t, "\x1b[1;31m", tok.Text)

		tok = l.Next()
		assert.Equal(t, CSIToken, tok.Type)
		assert.Equal(t, "\x1b[44m", tok.Text)

		tok = l.Next()
		assert.Equal(t, TextToken, tok.Type)
		assert.Equal(t, "Text", tok.Text)
	})

	t.Run("EmptyInput", func(t *testing.T) {
		l := NewLexer("")
		tok := l.Next()
		assert.Equal(t, EOFToken, tok.Type)
	})

	t.Run("OnlyCSI", func(t *testing.T) {
		l := NewLexer("\x1b[2J")

		tok := l.Next()
		assert.Equal(t, CSIToken, tok.Type)
		assert.Equal(t, "\x1b[2J", tok.Text)

		tok = l.Next()
		assert.Equal(t, EOFToken, tok.Type)
	})

	t.Run("ESCOnly", func(t *testing.T) {
		l := NewLexer("\x1b")

		tok := l.Next()
		assert.Equal(t, TextToken, tok.Type)
		assert.Equal(t, "\x1b", tok.Text)

		tok = l.Next()
		assert.Equal(t, EOFToken, tok.Type)
	})

	t.Run("ESCWithNonCSI", func(t *testing.T) {
		l := NewLexer("\x1bM")

		tok := l.Next()
		assert.Equal(t, ESCToken, tok.Type)
		assert.Equal(t, "\x1bM", tok.Text)

		tok = l.Next()
		assert.Equal(t, EOFToken, tok.Type)
	})
}

func TestParseCSI(t *testing.T) {
	t.Run("SimpleSGR", func(t *testing.T) {
		tok := Token{Type: CSIToken, Text: "\x1b[31m"}
		params, final := ParseCSI(tok)
		assert.Equal(t, "31", params)
		assert.Equal(t, byte('m'), final)
	})

	t.Run("MultipleParams", func(t *testing.T) {
		tok := Token{Type: CSIToken, Text: "\x1b[1;31;44m"}
		params, final := ParseCSI(tok)
		assert.Equal(t, "1;31;44", params)
		assert.Equal(t, byte('m'), final)
	})

	t.Run("NoParams", func(t *testing.T) {
		tok := Token{Type: CSIToken, Text: "\x1b[m"}
		params, final := ParseCSI(tok)
		assert.Equal(t, "", params)
		assert.Equal(t, byte('m'), final)
	})

	t.Run("CursorMovement", func(t *testing.T) {
		tok := Token{Type: CSIToken, Text: "\x1b[10;20H"}
		params, final := ParseCSI(tok)
		assert.Equal(t, "10;20", params)
		assert.Equal(t, byte('H'), final)
	})

	t.Run("NotCSI", func(t *testing.T) {
		tok := Token{Type: TextToken, Text: "hello"}
		params, final := ParseCSI(tok)
		assert.Equal(t, "", params)
		assert.Equal(t, byte(0), final)
	})
}

func TestParseSGRParams(t *testing.T) {
	t.Run("SingleParam", func(t *testing.T) {
		params := ParseSGRParams("31")
		assert.Equal(t, []int{31}, params)
	})

	t.Run("MultipleParams", func(t *testing.T) {
		params := ParseSGRParams("1;31;44")
		assert.Equal(t, []int{1, 31, 44}, params)
	})

	t.Run("Empty", func(t *testing.T) {
		params := ParseSGRParams("")
		assert.Equal(t, []int{0}, params)
	})

	t.Run("RGBColor", func(t *testing.T) {
		params := ParseSGRParams("38;2;255;128;64")
		assert.Equal(t, []int{38, 2, 255, 128, 64}, params)
	})

	t.Run("256Color", func(t *testing.T) {
		params := ParseSGRParams("38;5;196")
		assert.Equal(t, []int{38, 5, 196}, params)
	})

	t.Run("WithEmptyParts", func(t *testing.T) {
		params := ParseSGRParams("1;;31")
		assert.Equal(t, []int{1, 0, 31}, params)
	})
}
