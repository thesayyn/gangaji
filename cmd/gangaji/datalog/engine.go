package datalog

import (
	"fmt"
	"math"
	"sort"
)

// Engine evaluates Datalog programs
type Engine struct {
	facts    map[string][]Fact // predicate -> facts
	rules    []Rule
	builtins map[string]BuiltinFunc
}

// BuiltinFunc represents a built-in function
type BuiltinFunc func(args []interface{}) (interface{}, error)

// NewEngine creates a new Datalog engine
func NewEngine() *Engine {
	e := &Engine{
		facts:    make(map[string][]Fact),
		builtins: make(map[string]BuiltinFunc),
	}
	e.registerDefaultBuiltins()
	return e
}

// RegisterBuiltin registers a built-in function
func (e *Engine) RegisterBuiltin(name string, fn BuiltinFunc) {
	e.builtins[name] = fn
}

// AddFact adds a fact to the database
func (e *Engine) AddFact(f Fact) {
	e.facts[f.Predicate] = append(e.facts[f.Predicate], f)
}

// AddFacts adds multiple facts to the database
func (e *Engine) AddFacts(facts []Fact) {
	for _, f := range facts {
		e.AddFact(f)
	}
}

// AddRule adds a rule to the program
func (e *Engine) AddRule(r Rule) {
	e.rules = append(e.rules, r)
}

// AddRules adds multiple rules to the program
func (e *Engine) AddRules(rules []Rule) {
	e.rules = append(e.rules, rules...)
}

// LoadProgram loads rules from a parsed program
func (e *Engine) LoadProgram(program *Program) {
	e.AddRules(program.Rules)
}

// GetFacts returns all facts for a predicate
func (e *Engine) GetFacts(predicate string) []Fact {
	return e.facts[predicate]
}

// Evaluate runs the Datalog program until fixpoint
func (e *Engine) Evaluate() error {
	// Semi-naive bottom-up evaluation
	for {
		newFacts := 0

		for _, rule := range e.rules {
			derived, err := e.evaluateRule(rule)
			if err != nil {
				return err
			}

			for _, fact := range derived {
				if !e.factExists(fact) {
					e.AddFact(fact)
					newFacts++
				}
			}
		}

		// Fixpoint reached
		if newFacts == 0 {
			break
		}
	}

	return nil
}

// evaluateRule evaluates a single rule and returns derived facts
func (e *Engine) evaluateRule(rule Rule) ([]Fact, error) {
	// Find all bindings that satisfy the body
	bindings, err := e.evaluateBody(rule.Body, []Bindings{make(Bindings)})
	if err != nil {
		return nil, err
	}

	// Generate facts from bindings
	var facts []Fact
	for _, b := range bindings {
		fact, err := e.instantiateAtom(rule.Head, b)
		if err != nil {
			continue // Skip if can't instantiate
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

// evaluateBody evaluates the body clauses and returns satisfying bindings
func (e *Engine) evaluateBody(clauses []Clause, bindings []Bindings) ([]Bindings, error) {
	result := bindings

	for _, clause := range clauses {
		var newBindings []Bindings

		for _, b := range result {
			extended, err := e.evaluateClause(clause, b)
			if err != nil {
				return nil, err
			}
			newBindings = append(newBindings, extended...)
		}

		result = newBindings
		if len(result) == 0 {
			break
		}
	}

	return result, nil
}

// evaluateClause evaluates a single clause
func (e *Engine) evaluateClause(clause Clause, bindings Bindings) ([]Bindings, error) {
	switch c := clause.(type) {
	case AtomClause:
		return e.evaluateAtom(c.Atom, bindings)
	case Comparison:
		return e.evaluateComparison(c, bindings)
	case Assignment:
		return e.evaluateAssignment(c, bindings)
	case Aggregation:
		return e.evaluateAggregation(c, bindings)
	case Negation:
		return e.evaluateNegation(c, bindings)
	default:
		return nil, fmt.Errorf("unknown clause type: %T", clause)
	}
}

// evaluateAtom evaluates an atom against the fact database
func (e *Engine) evaluateAtom(atom Atom, bindings Bindings) ([]Bindings, error) {
	facts := e.facts[atom.Predicate]
	var result []Bindings

	for _, fact := range facts {
		if len(fact.Args) != len(atom.Args) {
			continue
		}

		newBindings := bindings.Clone()
		match := true

		for i, arg := range atom.Args {
			factVal := fact.Args[i]

			switch a := arg.(type) {
			case Variable:
				if existing, ok := newBindings[a]; ok {
					// Variable already bound - check equality
					if !valuesEqual(existing, factVal) {
						match = false
						break
					}
				} else {
					// Bind variable
					newBindings[a] = factVal
				}
			case Constant:
				if !valuesEqual(a.Value, factVal) {
					match = false
					break
				}
			case Wildcard:
				// Wildcard matches anything
			}
		}

		if match {
			result = append(result, newBindings)
		}
	}

	return result, nil
}

// evaluateComparison evaluates a comparison clause
func (e *Engine) evaluateComparison(comp Comparison, bindings Bindings) ([]Bindings, error) {
	leftVal, err := e.resolveTerm(comp.Left, bindings)
	if err != nil {
		return nil, nil // Can't resolve - no match
	}

	rightVal, err := e.resolveTerm(comp.Right, bindings)
	if err != nil {
		return nil, nil // Can't resolve - no match
	}

	result, err := compareValues(leftVal, rightVal, comp.Op)
	if err != nil {
		return nil, err
	}

	if result {
		return []Bindings{bindings}, nil
	}
	return nil, nil
}

// evaluateAssignment evaluates an assignment clause
func (e *Engine) evaluateAssignment(assign Assignment, bindings Bindings) ([]Bindings, error) {
	value, err := e.evaluateExpression(assign.Expr, bindings)
	if err != nil {
		return nil, nil // Can't evaluate - no match
	}

	newBindings := bindings.Clone()
	newBindings[assign.Variable] = value
	return []Bindings{newBindings}, nil
}

// evaluateAggregation evaluates an aggregation clause
func (e *Engine) evaluateAggregation(agg Aggregation, bindings Bindings) ([]Bindings, error) {
	// Find all bindings that satisfy the body
	bodyBindings, err := e.evaluateBody(agg.Body, []Bindings{bindings.Clone()})
	if err != nil {
		return nil, err
	}

	// Collect values to aggregate
	var values []float64
	for _, b := range bodyBindings {
		if agg.Op == AggCount {
			values = append(values, 1)
		} else {
			val, err := e.resolveTerm(agg.Variable, b)
			if err != nil {
				continue
			}
			numVal, err := toFloat64(val)
			if err != nil {
				continue
			}
			values = append(values, numVal)
		}
	}

	// Compute aggregate
	var result float64
	switch agg.Op {
	case AggCount:
		result = float64(len(values))
	case AggSum:
		for _, v := range values {
			result += v
		}
	case AggMax:
		if len(values) == 0 {
			return nil, nil
		}
		result = values[0]
		for _, v := range values[1:] {
			if v > result {
				result = v
			}
		}
	case AggMin:
		if len(values) == 0 {
			return nil, nil
		}
		result = values[0]
		for _, v := range values[1:] {
			if v < result {
				result = v
			}
		}
	case AggAvg:
		if len(values) == 0 {
			return nil, nil
		}
		for _, v := range values {
			result += v
		}
		result /= float64(len(values))
	}

	newBindings := bindings.Clone()
	newBindings[agg.Into] = result
	return []Bindings{newBindings}, nil
}

// evaluateNegation evaluates a negation-as-failure clause
func (e *Engine) evaluateNegation(neg Negation, bindings Bindings) ([]Bindings, error) {
	matches, err := e.evaluateAtom(neg.Atom, bindings)
	if err != nil {
		return nil, err
	}

	// Negation succeeds if no matches
	if len(matches) == 0 {
		return []Bindings{bindings}, nil
	}
	return nil, nil
}

// evaluateExpression evaluates an arithmetic expression
func (e *Engine) evaluateExpression(expr Expression, bindings Bindings) (interface{}, error) {
	switch ex := expr.(type) {
	case TermExpr:
		return e.resolveTerm(ex.Term, bindings)

	case BinaryExpr:
		leftVal, err := e.evaluateExpression(ex.Left, bindings)
		if err != nil {
			return nil, err
		}
		rightVal, err := e.evaluateExpression(ex.Right, bindings)
		if err != nil {
			return nil, err
		}

		left, err := toFloat64(leftVal)
		if err != nil {
			return nil, err
		}
		right, err := toFloat64(rightVal)
		if err != nil {
			return nil, err
		}

		switch ex.Op {
		case OpAdd:
			return left + right, nil
		case OpSub:
			return left - right, nil
		case OpMul:
			return left * right, nil
		case OpDiv:
			if right == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case OpMod:
			return math.Mod(left, right), nil
		default:
			return nil, fmt.Errorf("unknown operator: %s", ex.Op)
		}

	case FunctionCall:
		fn, ok := e.builtins[ex.Name]
		if !ok {
			return nil, fmt.Errorf("unknown function: %s", ex.Name)
		}

		args := make([]interface{}, len(ex.Args))
		for i, arg := range ex.Args {
			val, err := e.evaluateExpression(arg, bindings)
			if err != nil {
				return nil, err
			}
			args[i] = val
		}

		return fn(args)

	default:
		return nil, fmt.Errorf("unknown expression type: %T", expr)
	}
}

// resolveTerm resolves a term to its value given bindings
func (e *Engine) resolveTerm(term Term, bindings Bindings) (interface{}, error) {
	switch t := term.(type) {
	case Variable:
		if val, ok := bindings[t]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("unbound variable: %s", t)
	case Constant:
		return t.Value, nil
	case Wildcard:
		return nil, fmt.Errorf("cannot resolve wildcard")
	default:
		return nil, fmt.Errorf("unknown term type: %T", term)
	}
}

// instantiateAtom creates a fact from an atom and bindings
func (e *Engine) instantiateAtom(atom Atom, bindings Bindings) (Fact, error) {
	args := make([]interface{}, len(atom.Args))

	for i, arg := range atom.Args {
		val, err := e.resolveTerm(arg, bindings)
		if err != nil {
			return Fact{}, err
		}
		args[i] = val
	}

	return Fact{Predicate: atom.Predicate, Args: args}, nil
}

// factExists checks if a fact already exists in the database
func (e *Engine) factExists(fact Fact) bool {
	for _, f := range e.facts[fact.Predicate] {
		if factsEqual(f, fact) {
			return true
		}
	}
	return false
}

// EvaluateSuggestionRule evaluates a suggestion rule and returns matching bindings
func (e *Engine) EvaluateSuggestionRule(rule SuggestionRule) ([]Bindings, error) {
	return e.evaluateBody(rule.Conditions, []Bindings{make(Bindings)})
}

// Query queries the database for facts matching a pattern
func (e *Engine) Query(atom Atom) ([]Bindings, error) {
	return e.evaluateAtom(atom, make(Bindings))
}

// QueryOne queries for a single result
func (e *Engine) QueryOne(atom Atom) (Bindings, bool) {
	results, err := e.Query(atom)
	if err != nil || len(results) == 0 {
		return nil, false
	}
	return results[0], true
}

// FactCount returns the total number of facts
func (e *Engine) FactCount() int {
	count := 0
	for _, facts := range e.facts {
		count += len(facts)
	}
	return count
}

// PredicateNames returns all predicate names
func (e *Engine) PredicateNames() []string {
	names := make([]string, 0, len(e.facts))
	for name := range e.facts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Helper functions

func valuesEqual(a, b interface{}) bool {
	// Handle numeric comparisons
	aNum, aOk := toFloat64NoErr(a)
	bNum, bOk := toFloat64NoErr(b)
	if aOk && bOk {
		return aNum == bNum
	}

	// String comparison
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func factsEqual(a, b Fact) bool {
	if a.Predicate != b.Predicate || len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		if !valuesEqual(a.Args[i], b.Args[i]) {
			return false
		}
	}
	return true
}

func compareValues(left, right interface{}, op ComparisonOp) (bool, error) {
	// Try numeric comparison first
	leftNum, leftOk := toFloat64NoErr(left)
	rightNum, rightOk := toFloat64NoErr(right)

	if leftOk && rightOk {
		switch op {
		case OpEq:
			return leftNum == rightNum, nil
		case OpNeq:
			return leftNum != rightNum, nil
		case OpLt:
			return leftNum < rightNum, nil
		case OpLte:
			return leftNum <= rightNum, nil
		case OpGt:
			return leftNum > rightNum, nil
		case OpGte:
			return leftNum >= rightNum, nil
		}
	}

	// Fall back to string comparison
	leftStr := fmt.Sprint(left)
	rightStr := fmt.Sprint(right)

	switch op {
	case OpEq:
		return leftStr == rightStr, nil
	case OpNeq:
		return leftStr != rightStr, nil
	case OpLt:
		return leftStr < rightStr, nil
	case OpLte:
		return leftStr <= rightStr, nil
	case OpGt:
		return leftStr > rightStr, nil
	case OpGte:
		return leftStr >= rightStr, nil
	default:
		return false, fmt.Errorf("unknown comparison operator: %s", op)
	}
}

func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

func toFloat64NoErr(val interface{}) (float64, bool) {
	num, err := toFloat64(val)
	return num, err == nil
}

func (e *Engine) registerDefaultBuiltins() {
	// Mathematical functions
	e.RegisterBuiltin("abs", func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("abs expects 1 argument")
		}
		val, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		return math.Abs(val), nil
	})

	e.RegisterBuiltin("round", func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("round expects 1 argument")
		}
		val, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		return math.Round(val), nil
	})

	e.RegisterBuiltin("floor", func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("floor expects 1 argument")
		}
		val, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		return math.Floor(val), nil
	})

	e.RegisterBuiltin("ceil", func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("ceil expects 1 argument")
		}
		val, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		return math.Ceil(val), nil
	})
}
