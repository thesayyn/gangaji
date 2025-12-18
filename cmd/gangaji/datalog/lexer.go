package datalog

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError

	// Literals
	TokenIdent    // identifier (predicate name, etc.)
	TokenVariable // ?Name
	TokenString   // "string"
	TokenNumber   // 123, 45.6
	TokenWildcard // _

	// Keywords
	TokenRule       // rule
	TokenWhen       // when
	TokenThen       // then
	TokenSuggestion // suggestion
	TokenAggregate  // aggregate
	TokenNot        // not
	TokenCount      // count
	TokenSum        // sum
	TokenMax        // max
	TokenMin        // min
	TokenAvg        // avg

	// Operators
	TokenImplies   // :-
	TokenComma     // ,
	TokenDot       // .
	TokenLParen    // (
	TokenRParen    // )
	TokenLBracket  // [
	TokenRBracket  // ]
	TokenLBrace    // {
	TokenRBrace    // }
	TokenColon     // :
	TokenEq        // =
	TokenNeq       // !=
	TokenLt        // <
	TokenLte       // <=
	TokenGt        // >
	TokenGte       // >=
	TokenPlus      // +
	TokenMinus     // -
	TokenStar      // *
	TokenSlash     // /
	TokenPercent   // %
)

var tokenNames = map[TokenType]string{
	TokenEOF:        "EOF",
	TokenError:      "Error",
	TokenIdent:      "Ident",
	TokenVariable:   "Variable",
	TokenString:     "String",
	TokenNumber:     "Number",
	TokenWildcard:   "Wildcard",
	TokenRule:       "rule",
	TokenWhen:       "when",
	TokenThen:       "then",
	TokenSuggestion: "suggestion",
	TokenAggregate:  "aggregate",
	TokenNot:        "not",
	TokenCount:      "count",
	TokenSum:        "sum",
	TokenMax:        "max",
	TokenMin:        "min",
	TokenAvg:        "avg",
	TokenImplies:    ":-",
	TokenComma:      ",",
	TokenDot:        ".",
	TokenLParen:     "(",
	TokenRParen:     ")",
	TokenLBracket:   "[",
	TokenRBracket:   "]",
	TokenLBrace:     "{",
	TokenRBrace:     "}",
	TokenColon:      ":",
	TokenEq:         "=",
	TokenNeq:        "!=",
	TokenLt:         "<",
	TokenLte:        "<=",
	TokenGt:         ">",
	TokenGte:        ">=",
	TokenPlus:       "+",
	TokenMinus:      "-",
	TokenStar:       "*",
	TokenSlash:      "/",
	TokenPercent:    "%",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Token(%d)", t)
}

var keywords = map[string]TokenType{
	"rule":       TokenRule,
	"when":       TokenWhen,
	"then":       TokenThen,
	"suggestion": TokenSuggestion,
	"aggregate":  TokenAggregate,
	"not":        TokenNot,
	"count":      TokenCount,
	"sum":        TokenSum,
	"max":        TokenMax,
	"min":        TokenMin,
	"avg":        TokenAvg,
}

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Value   string
	Line    int
	Column  int
}

func (t Token) String() string {
	return fmt.Sprintf("Token{%s, %q, %d:%d}", t.Type, t.Value, t.Line, t.Column)
}

// Lexer tokenizes Datalog source code
type Lexer struct {
	input  string
	pos    int
	line   int
	column int
	tokens []Token
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
	}
}

// Tokenize returns all tokens from the input
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok := l.nextToken()
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
		if tok.Type == TokenError {
			return nil, fmt.Errorf("lexer error at %d:%d: %s", tok.Line, tok.Column, tok.Value)
		}
	}
	return l.tokens, nil
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return rune(l.input[l.pos])
}

func (l *Lexer) peekN(n int) rune {
	pos := l.pos + n
	if pos >= len(l.input) {
		return 0
	}
	return rune(l.input[pos])
}

func (l *Lexer) advance() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := rune(l.input[l.pos])
	l.pos++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

func (l *Lexer) skipWhitespace() {
	for {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.advance()
		} else if ch == '%' {
			// Skip comment to end of line
			for l.peek() != '\n' && l.peek() != 0 {
				l.advance()
			}
		} else {
			break
		}
	}
}

func (l *Lexer) makeToken(typ TokenType, value string) Token {
	return Token{
		Type:   typ,
		Value:  value,
		Line:   l.line,
		Column: l.column - len(value),
	}
}

func (l *Lexer) nextToken() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return l.makeToken(TokenEOF, "")
	}

	ch := l.peek()
	startLine := l.line
	startCol := l.column

	// Single character tokens
	switch ch {
	case '(':
		l.advance()
		return Token{TokenLParen, "(", startLine, startCol}
	case ')':
		l.advance()
		return Token{TokenRParen, ")", startLine, startCol}
	case '[':
		l.advance()
		return Token{TokenLBracket, "[", startLine, startCol}
	case ']':
		l.advance()
		return Token{TokenRBracket, "]", startLine, startCol}
	case '{':
		l.advance()
		return Token{TokenLBrace, "{", startLine, startCol}
	case '}':
		l.advance()
		return Token{TokenRBrace, "}", startLine, startCol}
	case ',':
		l.advance()
		return Token{TokenComma, ",", startLine, startCol}
	case '.':
		l.advance()
		return Token{TokenDot, ".", startLine, startCol}
	case '+':
		l.advance()
		return Token{TokenPlus, "+", startLine, startCol}
	case '*':
		l.advance()
		return Token{TokenStar, "*", startLine, startCol}
	case '/':
		l.advance()
		return Token{TokenSlash, "/", startLine, startCol}
	}

	// Two character tokens
	if ch == ':' {
		l.advance()
		if l.peek() == '-' {
			l.advance()
			return Token{TokenImplies, ":-", startLine, startCol}
		}
		return Token{TokenColon, ":", startLine, startCol}
	}

	if ch == '!' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{TokenNeq, "!=", startLine, startCol}
		}
		return Token{TokenError, "unexpected '!'", startLine, startCol}
	}

	if ch == '<' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{TokenLte, "<=", startLine, startCol}
		}
		return Token{TokenLt, "<", startLine, startCol}
	}

	if ch == '>' {
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return Token{TokenGte, ">=", startLine, startCol}
		}
		return Token{TokenGt, ">", startLine, startCol}
	}

	if ch == '=' {
		l.advance()
		return Token{TokenEq, "=", startLine, startCol}
	}

	if ch == '-' {
		l.advance()
		// Check if it's a negative number
		if unicode.IsDigit(l.peek()) {
			return l.scanNumber("-")
		}
		return Token{TokenMinus, "-", startLine, startCol}
	}

	// Wildcard
	if ch == '_' && !isIdentChar(l.peekN(1)) {
		l.advance()
		return Token{TokenWildcard, "_", startLine, startCol}
	}

	// Variable (?Name)
	if ch == '?' {
		l.advance()
		return l.scanVariable()
	}

	// String literal
	if ch == '"' {
		return l.scanString()
	}

	// Number
	if unicode.IsDigit(ch) {
		return l.scanNumber("")
	}

	// Identifier or keyword
	if isIdentStart(ch) {
		return l.scanIdentifier()
	}

	l.advance()
	return Token{TokenError, fmt.Sprintf("unexpected character '%c'", ch), startLine, startCol}
}

func (l *Lexer) scanVariable() Token {
	startLine := l.line
	startCol := l.column - 1 // account for '?'

	var sb strings.Builder
	sb.WriteRune('?')

	for isIdentChar(l.peek()) {
		sb.WriteRune(l.advance())
	}

	return Token{TokenVariable, sb.String(), startLine, startCol}
}

func (l *Lexer) scanString() Token {
	startLine := l.line
	startCol := l.column

	l.advance() // consume opening quote

	var sb strings.Builder
	for {
		ch := l.peek()
		if ch == 0 {
			return Token{TokenError, "unterminated string", startLine, startCol}
		}
		if ch == '"' {
			l.advance()
			break
		}
		if ch == '\\' {
			l.advance()
			escaped := l.advance()
			switch escaped {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			case '"':
				sb.WriteRune('"')
			case '\\':
				sb.WriteRune('\\')
			default:
				sb.WriteRune(escaped)
			}
		} else {
			sb.WriteRune(l.advance())
		}
	}

	return Token{TokenString, sb.String(), startLine, startCol}
}

func (l *Lexer) scanNumber(prefix string) Token {
	startLine := l.line
	startCol := l.column

	var sb strings.Builder
	sb.WriteString(prefix)

	for unicode.IsDigit(l.peek()) {
		sb.WriteRune(l.advance())
	}

	// Check for decimal point
	if l.peek() == '.' && unicode.IsDigit(l.peekN(1)) {
		sb.WriteRune(l.advance()) // consume '.'
		for unicode.IsDigit(l.peek()) {
			sb.WriteRune(l.advance())
		}
	}

	return Token{TokenNumber, sb.String(), startLine, startCol}
}

func (l *Lexer) scanIdentifier() Token {
	startLine := l.line
	startCol := l.column

	var sb strings.Builder
	for isIdentChar(l.peek()) {
		sb.WriteRune(l.advance())
	}

	value := sb.String()

	// Check if it's a keyword
	if typ, ok := keywords[value]; ok {
		return Token{typ, value, startLine, startCol}
	}

	return Token{TokenIdent, value, startLine, startCol}
}

func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}
