package datalog

import (
	"fmt"
	"strconv"
)

// Parser parses Datalog source code into an AST
type Parser struct {
	tokens []Token
	pos    int
}

// NewParser creates a new parser for the given tokens
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

// Parse parses the input and returns a Program
func Parse(input string) (*Program, error) {
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens)
	return parser.ParseProgram()
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peekN(n int) Token {
	pos := p.pos + n
	if pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[pos]
}

func (p *Parser) advance() Token {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *Parser) expect(typ TokenType) (Token, error) {
	tok := p.peek()
	if tok.Type != typ {
		return tok, fmt.Errorf("expected %s, got %s at %d:%d", typ, tok.Type, tok.Line, tok.Column)
	}
	return p.advance(), nil
}

func (p *Parser) match(typ TokenType) bool {
	if p.peek().Type == typ {
		p.advance()
		return true
	}
	return false
}

// ParseProgram parses a complete Datalog program
func (p *Parser) ParseProgram() (*Program, error) {
	program := &Program{}

	for p.peek().Type != TokenEOF {
		if p.peek().Type == TokenRule {
			rule, err := p.parseSuggestionRule()
			if err != nil {
				return nil, err
			}
			program.SuggestionRules = append(program.SuggestionRules, rule)
		} else if p.peek().Type == TokenIdent || p.peek().Type == TokenVariable {
			rule, err := p.parseRule()
			if err != nil {
				return nil, err
			}
			program.Rules = append(program.Rules, rule)
		} else {
			tok := p.peek()
			return nil, fmt.Errorf("unexpected token %s at %d:%d", tok.Type, tok.Line, tok.Column)
		}
	}

	return program, nil
}

// parseRule parses a Datalog rule (head :- body.)
func (p *Parser) parseRule() (Rule, error) {
	head, err := p.parseAtom()
	if err != nil {
		return Rule{}, err
	}

	var body []Clause

	if p.match(TokenImplies) {
		body, err = p.parseBody()
		if err != nil {
			return Rule{}, err
		}
	}

	if _, err := p.expect(TokenDot); err != nil {
		return Rule{}, err
	}

	return Rule{Head: head, Body: body}, nil
}

// parseSuggestionRule parses a suggestion rule (rule name { when: ... then: ... })
func (p *Parser) parseSuggestionRule() (SuggestionRule, error) {
	if _, err := p.expect(TokenRule); err != nil {
		return SuggestionRule{}, err
	}

	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return SuggestionRule{}, err
	}

	if _, err := p.expect(TokenLBrace); err != nil {
		return SuggestionRule{}, err
	}

	rule := SuggestionRule{
		ID:   nameTok.Value,
		Name: nameTok.Value,
	}

	// Parse when: block
	if _, err := p.expect(TokenWhen); err != nil {
		return SuggestionRule{}, err
	}
	if _, err := p.expect(TokenColon); err != nil {
		return SuggestionRule{}, err
	}

	conditions, err := p.parseBody()
	if err != nil {
		return SuggestionRule{}, err
	}
	rule.Conditions = conditions

	if _, err := p.expect(TokenDot); err != nil {
		return SuggestionRule{}, err
	}

	// Parse then: block
	if _, err := p.expect(TokenThen); err != nil {
		return SuggestionRule{}, err
	}
	if _, err := p.expect(TokenColon); err != nil {
		return SuggestionRule{}, err
	}

	suggestion, err := p.parseSuggestionTemplate()
	if err != nil {
		return SuggestionRule{}, err
	}
	rule.Suggestion = suggestion

	if _, err := p.expect(TokenDot); err != nil {
		return SuggestionRule{}, err
	}

	if _, err := p.expect(TokenRBrace); err != nil {
		return SuggestionRule{}, err
	}

	return rule, nil
}

// parseSuggestionTemplate parses a suggestion(...) template
func (p *Parser) parseSuggestionTemplate() (SuggestionTemplate, error) {
	if _, err := p.expect(TokenSuggestion); err != nil {
		return SuggestionTemplate{}, err
	}

	if _, err := p.expect(TokenLParen); err != nil {
		return SuggestionTemplate{}, err
	}

	// Parse type (warning, info, success)
	typeTok, err := p.expect(TokenIdent)
	if err != nil {
		return SuggestionTemplate{}, err
	}

	if _, err := p.expect(TokenComma); err != nil {
		return SuggestionTemplate{}, err
	}

	// Parse impact (high, medium, low)
	impactTok, err := p.expect(TokenIdent)
	if err != nil {
		return SuggestionTemplate{}, err
	}

	if _, err := p.expect(TokenComma); err != nil {
		return SuggestionTemplate{}, err
	}

	// Parse title (string)
	titleTok, err := p.expect(TokenString)
	if err != nil {
		return SuggestionTemplate{}, err
	}

	if _, err := p.expect(TokenComma); err != nil {
		return SuggestionTemplate{}, err
	}

	// Parse body (string)
	bodyTok, err := p.expect(TokenString)
	if err != nil {
		return SuggestionTemplate{}, err
	}

	template := SuggestionTemplate{
		Type:   typeTok.Value,
		Impact: impactTok.Value,
		Title:  titleTok.Value,
		Body:   bodyTok.Value,
	}

	// Optional: target and metrics
	if p.match(TokenComma) {
		// Parse target
		if p.peek().Type == TokenString {
			targetTok := p.advance()
			template.Target = targetTok.Value
		} else if p.peek().Type == TokenVariable {
			varTok := p.advance()
			template.Target = varTok.Value
		}

		// Optional: metrics array
		if p.match(TokenComma) {
			metrics, err := p.parseMetricsArray()
			if err != nil {
				return SuggestionTemplate{}, err
			}
			template.Metrics = metrics
		}
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return SuggestionTemplate{}, err
	}

	return template, nil
}

// parseMetricsArray parses [[label, value], ...]
func (p *Parser) parseMetricsArray() ([]MetricTemplate, error) {
	if _, err := p.expect(TokenLBracket); err != nil {
		return nil, err
	}

	var metrics []MetricTemplate

	for p.peek().Type != TokenRBracket {
		if _, err := p.expect(TokenLBracket); err != nil {
			return nil, err
		}

		// Label
		labelTok, err := p.expect(TokenString)
		if err != nil {
			return nil, err
		}

		if _, err := p.expect(TokenComma); err != nil {
			return nil, err
		}

		// Value (string or expression)
		var value string
		if p.peek().Type == TokenString {
			valueTok := p.advance()
			value = valueTok.Value
		} else {
			// Parse as expression and convert to string representation
			expr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			value = expr.String()
		}

		if _, err := p.expect(TokenRBracket); err != nil {
			return nil, err
		}

		metrics = append(metrics, MetricTemplate{
			Label: labelTok.Value,
			Value: value,
		})

		if !p.match(TokenComma) {
			break
		}
	}

	if _, err := p.expect(TokenRBracket); err != nil {
		return nil, err
	}

	return metrics, nil
}

// parseBody parses a comma-separated list of clauses
func (p *Parser) parseBody() ([]Clause, error) {
	var clauses []Clause

	for {
		clause, err := p.parseClause()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, clause)

		if !p.match(TokenComma) {
			break
		}
	}

	return clauses, nil
}

// parseClause parses a single clause (atom, comparison, assignment, aggregation, negation)
func (p *Parser) parseClause() (Clause, error) {
	// Check for negation
	if p.match(TokenNot) {
		atom, err := p.parseAtom()
		if err != nil {
			return nil, err
		}
		return Negation{Atom: atom}, nil
	}

	// Check for aggregate
	if p.peek().Type == TokenAggregate {
		return p.parseAggregation()
	}

	// Check for variable assignment or comparison
	if p.peek().Type == TokenVariable {
		// Look ahead to see if this is an assignment or comparison
		if p.peekN(1).Type == TokenEq {
			// Could be assignment (expr) or comparison (term)
			return p.parseAssignmentOrComparison()
		}
		if isComparisonOp(p.peekN(1).Type) {
			return p.parseComparison()
		}
	}

	// Must be an atom
	atom, err := p.parseAtom()
	if err != nil {
		return nil, err
	}
	return AtomClause{Atom: atom}, nil
}

// parseAssignmentOrComparison parses either an assignment or a comparison starting with a variable
func (p *Parser) parseAssignmentOrComparison() (Clause, error) {
	varTok := p.advance() // consume variable
	variable := Variable(varTok.Value)

	p.advance() // consume '='

	// Try to parse as expression
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// If expression is just a term, treat as comparison
	if termExpr, ok := expr.(TermExpr); ok {
		return Comparison{
			Left:  variable,
			Op:    OpEq,
			Right: termExpr.Term,
		}, nil
	}

	// Otherwise it's an assignment
	return Assignment{
		Variable: variable,
		Expr:     expr,
	}, nil
}

// parseComparison parses a comparison (e.g., ?Pct > 10)
func (p *Parser) parseComparison() (Comparison, error) {
	left, err := p.parseTerm()
	if err != nil {
		return Comparison{}, err
	}

	opTok := p.advance()
	op, err := tokenToComparisonOp(opTok.Type)
	if err != nil {
		return Comparison{}, err
	}

	right, err := p.parseTerm()
	if err != nil {
		return Comparison{}, err
	}

	return Comparison{Left: left, Op: op, Right: right}, nil
}

// parseAggregation parses an aggregation clause
func (p *Parser) parseAggregation() (Aggregation, error) {
	if _, err := p.expect(TokenAggregate); err != nil {
		return Aggregation{}, err
	}

	if _, err := p.expect(TokenLParen); err != nil {
		return Aggregation{}, err
	}

	// Parse aggregate operation (count, sum, max, min, avg)
	opTok := p.advance()
	op, err := tokenToAggregateOp(opTok.Type)
	if err != nil {
		// Try as identifier
		if opTok.Type == TokenIdent {
			switch opTok.Value {
			case "count":
				op = AggCount
			case "sum":
				op = AggSum
			case "max":
				op = AggMax
			case "min":
				op = AggMin
			case "avg":
				op = AggAvg
			default:
				return Aggregation{}, fmt.Errorf("unknown aggregate operation: %s", opTok.Value)
			}
		} else {
			return Aggregation{}, err
		}
	}

	var aggVar Variable

	// For count, no variable needed; for others, parse variable
	if op != AggCount {
		if _, err := p.expect(TokenLParen); err != nil {
			return Aggregation{}, err
		}
		varTok, err := p.expect(TokenVariable)
		if err != nil {
			return Aggregation{}, err
		}
		aggVar = Variable(varTok.Value)
		if _, err := p.expect(TokenRParen); err != nil {
			return Aggregation{}, err
		}
	}

	if _, err := p.expect(TokenComma); err != nil {
		return Aggregation{}, err
	}

	// Parse body clauses
	body, err := p.parseBody()
	if err != nil {
		return Aggregation{}, err
	}

	if _, err := p.expect(TokenComma); err != nil {
		return Aggregation{}, err
	}

	// Parse result variable
	resultTok, err := p.expect(TokenVariable)
	if err != nil {
		return Aggregation{}, err
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return Aggregation{}, err
	}

	return Aggregation{
		Op:       op,
		Variable: aggVar,
		Body:     body,
		Into:     Variable(resultTok.Value),
	}, nil
}

// parseAtom parses an atom (predicate with arguments)
func (p *Parser) parseAtom() (Atom, error) {
	predTok, err := p.expect(TokenIdent)
	if err != nil {
		return Atom{}, err
	}

	if _, err := p.expect(TokenLParen); err != nil {
		return Atom{}, err
	}

	var args []Term
	for p.peek().Type != TokenRParen {
		term, err := p.parseTerm()
		if err != nil {
			return Atom{}, err
		}
		args = append(args, term)

		if !p.match(TokenComma) {
			break
		}
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return Atom{}, err
	}

	return Atom{Predicate: predTok.Value, Args: args}, nil
}

// parseTerm parses a term (variable, constant, or wildcard)
func (p *Parser) parseTerm() (Term, error) {
	tok := p.peek()

	switch tok.Type {
	case TokenVariable:
		p.advance()
		return Variable(tok.Value), nil

	case TokenWildcard:
		p.advance()
		return Wildcard{}, nil

	case TokenString:
		p.advance()
		return Constant{Value: tok.Value}, nil

	case TokenNumber:
		p.advance()
		if val, err := strconv.ParseInt(tok.Value, 10, 64); err == nil {
			return Constant{Value: val}, nil
		}
		if val, err := strconv.ParseFloat(tok.Value, 64); err == nil {
			return Constant{Value: val}, nil
		}
		return nil, fmt.Errorf("invalid number: %s", tok.Value)

	case TokenIdent:
		// Could be a boolean or identifier constant
		p.advance()
		switch tok.Value {
		case "true":
			return Constant{Value: true}, nil
		case "false":
			return Constant{Value: false}, nil
		default:
			return Constant{Value: tok.Value}, nil
		}

	default:
		return nil, fmt.Errorf("expected term, got %s at %d:%d", tok.Type, tok.Line, tok.Column)
	}
}

// parseExpression parses an arithmetic expression
func (p *Parser) parseExpression() (Expression, error) {
	return p.parseAdditive()
}

func (p *Parser) parseAdditive() (Expression, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.peek()
		var op ArithOp
		switch tok.Type {
		case TokenPlus:
			op = OpAdd
		case TokenMinus:
			op = OpSub
		default:
			return left, nil
		}
		p.advance()

		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Left: left, Op: op, Right: right}
	}
}

func (p *Parser) parseMultiplicative() (Expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.peek()
		var op ArithOp
		switch tok.Type {
		case TokenStar:
			op = OpMul
		case TokenSlash:
			op = OpDiv
		case TokenPercent:
			op = OpMod
		default:
			return left, nil
		}
		p.advance()

		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Left: left, Op: op, Right: right}
	}
}

func (p *Parser) parseUnary() (Expression, error) {
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (Expression, error) {
	tok := p.peek()

	// Parenthesized expression
	if tok.Type == TokenLParen {
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil
	}

	// Function call
	if tok.Type == TokenIdent && p.peekN(1).Type == TokenLParen {
		return p.parseFunctionCall()
	}

	// Term
	term, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	return TermExpr{Term: term}, nil
}

func (p *Parser) parseFunctionCall() (FunctionCall, error) {
	nameTok := p.advance()

	if _, err := p.expect(TokenLParen); err != nil {
		return FunctionCall{}, err
	}

	var args []Expression
	for p.peek().Type != TokenRParen {
		expr, err := p.parseExpression()
		if err != nil {
			return FunctionCall{}, err
		}
		args = append(args, expr)

		if !p.match(TokenComma) {
			break
		}
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return FunctionCall{}, err
	}

	return FunctionCall{Name: nameTok.Value, Args: args}, nil
}

func isComparisonOp(typ TokenType) bool {
	switch typ {
	case TokenEq, TokenNeq, TokenLt, TokenLte, TokenGt, TokenGte:
		return true
	}
	return false
}

func tokenToComparisonOp(typ TokenType) (ComparisonOp, error) {
	switch typ {
	case TokenEq:
		return OpEq, nil
	case TokenNeq:
		return OpNeq, nil
	case TokenLt:
		return OpLt, nil
	case TokenLte:
		return OpLte, nil
	case TokenGt:
		return OpGt, nil
	case TokenGte:
		return OpGte, nil
	default:
		return "", fmt.Errorf("expected comparison operator, got %s", typ)
	}
}

func tokenToAggregateOp(typ TokenType) (AggregateOp, error) {
	switch typ {
	case TokenCount:
		return AggCount, nil
	case TokenSum:
		return AggSum, nil
	case TokenMax:
		return AggMax, nil
	case TokenMin:
		return AggMin, nil
	case TokenAvg:
		return AggAvg, nil
	default:
		return "", fmt.Errorf("expected aggregate operator, got %s", typ)
	}
}
