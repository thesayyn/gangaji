package datalog

import (
	"fmt"
	"strings"
)

// Term represents a term in Datalog (variable or constant)
type Term interface {
	isTerm()
	String() string
}

// Variable represents a Datalog variable (e.g., ?Name, ?Dur)
type Variable string

func (v Variable) isTerm()        {}
func (v Variable) String() string { return string(v) }

// Constant represents a constant value
type Constant struct {
	Value interface{} // string, int64, float64, bool
}

func (c Constant) isTerm() {}
func (c Constant) String() string {
	switch v := c.Value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprint(v)
	}
}

// Wildcard represents an anonymous variable (_)
type Wildcard struct{}

func (w Wildcard) isTerm()        {}
func (w Wildcard) String() string { return "_" }

// Atom represents a predicate with arguments (e.g., trace_event(?E, ?Name, _, _, ?Dur))
type Atom struct {
	Predicate string
	Args      []Term
}

func (a Atom) String() string {
	args := make([]string, len(a.Args))
	for i, arg := range a.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", a.Predicate, strings.Join(args, ", "))
}

// Clause represents a clause in a rule body
type Clause interface {
	isClause()
	String() string
}

// AtomClause wraps an Atom as a Clause
type AtomClause struct {
	Atom Atom
}

func (a AtomClause) isClause()      {}
func (a AtomClause) String() string { return a.Atom.String() }

// Comparison represents a comparison (e.g., ?Pct > 10)
type Comparison struct {
	Left  Term
	Op    ComparisonOp
	Right Term
}

type ComparisonOp string

const (
	OpEq  ComparisonOp = "="
	OpNeq ComparisonOp = "!="
	OpLt  ComparisonOp = "<"
	OpLte ComparisonOp = "<="
	OpGt  ComparisonOp = ">"
	OpGte ComparisonOp = ">="
)

func (c Comparison) isClause() {}
func (c Comparison) String() string {
	return fmt.Sprintf("%s %s %s", c.Left.String(), c.Op, c.Right.String())
}

// Assignment represents an arithmetic assignment (e.g., ?Pct = (?Dur * 100) / ?Total)
type Assignment struct {
	Variable Variable
	Expr     Expression
}

func (a Assignment) isClause() {}
func (a Assignment) String() string {
	return fmt.Sprintf("%s = %s", a.Variable, a.Expr.String())
}

// Expression represents an arithmetic expression
type Expression interface {
	isExpr()
	String() string
}

// TermExpr wraps a Term as an Expression
type TermExpr struct {
	Term Term
}

func (t TermExpr) isExpr()       {}
func (t TermExpr) String() string { return t.Term.String() }

// BinaryExpr represents a binary arithmetic expression
type BinaryExpr struct {
	Left  Expression
	Op    ArithOp
	Right Expression
}

type ArithOp string

const (
	OpAdd ArithOp = "+"
	OpSub ArithOp = "-"
	OpMul ArithOp = "*"
	OpDiv ArithOp = "/"
	OpMod ArithOp = "%"
)

func (b BinaryExpr) isExpr() {}
func (b BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.String(), b.Op, b.Right.String())
}

// FunctionCall represents a built-in function call (e.g., format_time(?Dur))
type FunctionCall struct {
	Name string
	Args []Expression
}

func (f FunctionCall) isExpr() {}
func (f FunctionCall) String() string {
	args := make([]string, len(f.Args))
	for i, arg := range f.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", f.Name, strings.Join(args, ", "))
}

// Aggregation represents an aggregation (e.g., aggregate(sum(?Dur), ...))
type Aggregation struct {
	Op       AggregateOp
	Variable Variable   // Variable to aggregate (e.g., ?Dur for sum(?Dur))
	Body     []Clause   // Clauses to aggregate over
	Into     Variable   // Result variable
}

type AggregateOp string

const (
	AggCount AggregateOp = "count"
	AggSum   AggregateOp = "sum"
	AggMax   AggregateOp = "max"
	AggMin   AggregateOp = "min"
	AggAvg   AggregateOp = "avg"
)

func (a Aggregation) isClause() {}
func (a Aggregation) String() string {
	body := make([]string, len(a.Body))
	for i, c := range a.Body {
		body[i] = c.String()
	}
	return fmt.Sprintf("aggregate(%s(%s), %s, %s)", a.Op, a.Variable, strings.Join(body, ", "), a.Into)
}

// Negation represents negation-as-failure (not predicate(...))
type Negation struct {
	Atom Atom
}

func (n Negation) isClause() {}
func (n Negation) String() string {
	return fmt.Sprintf("not %s", n.Atom.String())
}

// Rule represents a Datalog rule (head :- body)
type Rule struct {
	Head Atom
	Body []Clause
}

func (r Rule) String() string {
	if len(r.Body) == 0 {
		return r.Head.String() + "."
	}
	body := make([]string, len(r.Body))
	for i, c := range r.Body {
		body[i] = c.String()
	}
	return fmt.Sprintf("%s :- %s.", r.Head.String(), strings.Join(body, ", "))
}

// Fact represents a ground fact (no variables)
type Fact struct {
	Predicate string
	Args      []interface{}
}

func (f Fact) String() string {
	args := make([]string, len(f.Args))
	for i, arg := range f.Args {
		switch v := arg.(type) {
		case string:
			args[i] = fmt.Sprintf("%q", v)
		default:
			args[i] = fmt.Sprint(v)
		}
	}
	return fmt.Sprintf("%s(%s)", f.Predicate, strings.Join(args, ", "))
}

// SuggestionRule represents a rule that generates suggestions
type SuggestionRule struct {
	ID         string
	Name       string
	Conditions []Clause
	Suggestion SuggestionTemplate
}

// SuggestionTemplate represents the output template for a suggestion
type SuggestionTemplate struct {
	Type    string            // "warning", "info", "success"
	Impact  string            // "high", "medium", "low"
	Title   string            // Template string with {Var} placeholders
	Body    string            // Template string with {Var} placeholders
	Target  string            // Template string with {Var} placeholders
	Metrics []MetricTemplate  // Metrics to display
}

// MetricTemplate represents a metric in a suggestion
type MetricTemplate struct {
	Label string // Template string
	Value string // Template string or expression
}

// Suggestion represents a generated suggestion
type Suggestion struct {
	ID       string   `json:"id"`
	RuleID   string   `json:"ruleId"`
	Type     string   `json:"type"`
	Impact   string   `json:"impact"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Target   string   `json:"target"`
	Metrics  []Metric `json:"metrics"`
}

// Metric represents a metric in a generated suggestion
type Metric struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Bindings represents variable bindings during evaluation
type Bindings map[Variable]interface{}

// Clone creates a copy of the bindings
func (b Bindings) Clone() Bindings {
	clone := make(Bindings, len(b))
	for k, v := range b {
		clone[k] = v
	}
	return clone
}

// Get returns the value bound to a variable, or nil if unbound
func (b Bindings) Get(v Variable) interface{} {
	return b[v]
}

// Set binds a variable to a value
func (b Bindings) Set(v Variable, val interface{}) {
	b[v] = val
}

// Program represents a complete Datalog program
type Program struct {
	Rules           []Rule           // Derived relation rules
	SuggestionRules []SuggestionRule // Rules that generate suggestions
}
