package suggestions

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/thesayyn/gangaji/cmd/gangaji/datalog"
)

//go:embed rules/*.dl rules/builtin/*.dl
var builtinRulesFS embed.FS

// Evaluator evaluates rules and generates suggestions
type Evaluator struct {
	engine   *datalog.Engine
	program  *datalog.Program
	rulesDir string // Optional external rules directory
}

// SuggestionsResult contains the evaluation results
type SuggestionsResult struct {
	Suggestions     []datalog.Suggestion `json:"suggestions"`
	RulesEvaluated  int                  `json:"rulesEvaluated"`
	FactsGenerated  int                  `json:"factsGenerated"`
	EvaluationTimeMs int64              `json:"evaluationTimeMs"`
}

// NewEvaluator creates a new evaluator
func NewEvaluator(rulesDir string) *Evaluator {
	engine := datalog.NewEngine()
	engine.RegisterFormattingBuiltins()

	return &Evaluator{
		engine:   engine,
		program:  &datalog.Program{},
		rulesDir: rulesDir,
	}
}

// LoadRules loads all rules from embedded and external sources
func (e *Evaluator) LoadRules() error {
	// Load embedded rules
	if err := e.loadEmbeddedRules(); err != nil {
		return fmt.Errorf("failed to load embedded rules: %w", err)
	}

	// Load external rules if specified
	if e.rulesDir != "" {
		if err := e.loadExternalRules(); err != nil {
			return fmt.Errorf("failed to load external rules: %w", err)
		}
	}

	// Load derived rules into engine
	e.engine.LoadProgram(e.program)

	return nil
}

// loadEmbeddedRules loads rules from embedded filesystem
func (e *Evaluator) loadEmbeddedRules() error {
	return fs.WalkDir(builtinRulesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".dl") {
			return nil
		}

		content, err := builtinRulesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		program, err := datalog.Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		e.program.Rules = append(e.program.Rules, program.Rules...)
		e.program.SuggestionRules = append(e.program.SuggestionRules, program.SuggestionRules...)

		return nil
	})
}

// loadExternalRules loads rules from external directory
func (e *Evaluator) loadExternalRules() error {
	return filepath.Walk(e.rulesDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".dl") {
			return nil
		}

		// Read and parse external rule file
		// Note: In a real implementation, you'd use os.ReadFile here
		// For now, external rules would need to be loaded differently
		return nil
	})
}

// Evaluate evaluates all rules against the provided trace events
func (e *Evaluator) Evaluate(events []datalog.TraceEvent) (*SuggestionsResult, error) {
	startTime := time.Now()

	// Generate facts from trace events
	facts := datalog.GenerateFacts(events)
	e.engine.AddFacts(facts)

	// Add event percentage facts
	var totalDuration float64
	for _, f := range facts {
		if f.Predicate == "total_duration" && len(f.Args) > 0 {
			if dur, ok := f.Args[0].(float64); ok {
				totalDuration = dur
			}
		}
	}
	percentFacts := datalog.GenerateEventPercentFacts(events, totalDuration)
	e.engine.AddFacts(percentFacts)

	// Evaluate derived rules to generate additional facts
	if err := e.engine.Evaluate(); err != nil {
		return nil, fmt.Errorf("failed to evaluate rules: %w", err)
	}

	// Evaluate suggestion rules
	var suggestions []datalog.Suggestion
	for _, rule := range e.program.SuggestionRules {
		bindings, err := e.engine.EvaluateSuggestionRule(rule)
		if err != nil {
			continue // Skip rules that fail
		}

		for _, b := range bindings {
			suggestion := e.generateSuggestion(rule, b)
			suggestions = append(suggestions, suggestion)
		}
	}

	// Sort suggestions by impact (high first)
	sort.Slice(suggestions, func(i, j int) bool {
		return impactOrder(suggestions[i].Impact) < impactOrder(suggestions[j].Impact)
	})

	// Deduplicate suggestions
	suggestions = deduplicateSuggestions(suggestions)

	return &SuggestionsResult{
		Suggestions:      suggestions,
		RulesEvaluated:   len(e.program.SuggestionRules),
		FactsGenerated:   e.engine.FactCount(),
		EvaluationTimeMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// generateSuggestion generates a suggestion from a rule and bindings
func (e *Evaluator) generateSuggestion(rule datalog.SuggestionRule, bindings datalog.Bindings) datalog.Suggestion {
	suggestion := datalog.Suggestion{
		ID:     fmt.Sprintf("%s-%d", rule.ID, time.Now().UnixNano()),
		RuleID: rule.ID,
		Type:   rule.Suggestion.Type,
		Impact: rule.Suggestion.Impact,
		Title:  renderTemplate(rule.Suggestion.Title, bindings),
		Body:   renderTemplate(rule.Suggestion.Body, bindings),
		Target: renderTemplate(rule.Suggestion.Target, bindings),
	}

	// Render metrics
	for _, m := range rule.Suggestion.Metrics {
		metric := datalog.Metric{
			Label: renderTemplate(m.Label, bindings),
			Value: renderMetricValue(m.Value, bindings),
		}
		suggestion.Metrics = append(suggestion.Metrics, metric)
	}

	return suggestion
}

// renderTemplate replaces {VarName} placeholders and bare ?Var with bound values
func renderTemplate(template string, bindings datalog.Bindings) string {
	result := template

	// Replace {VarName} patterns
	re := regexp.MustCompile(`\{(\??\w+)\}`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		varName := match[1 : len(match)-1]
		if !strings.HasPrefix(varName, "?") {
			varName = "?" + varName
		}

		if val, ok := bindings[datalog.Variable(varName)]; ok {
			return formatValue(val)
		}
		return match
	})

	// Also replace bare ?VarName patterns (when template is just a variable)
	bareVarRe := regexp.MustCompile(`^\?(\w+)$`)
	if bareVarRe.MatchString(result) {
		varName := result
		if val, ok := bindings[datalog.Variable(varName)]; ok {
			return formatValue(val)
		}
	}

	return result
}

// renderMetricValue renders a metric value, potentially calling functions
func renderMetricValue(value string, bindings datalog.Bindings) string {
	// Check if it's a function call like format_time(?Dur)
	if strings.HasPrefix(value, "format_time(") {
		varMatch := regexp.MustCompile(`format_time\((\?\w+)\)`).FindStringSubmatch(value)
		if len(varMatch) > 1 {
			varName := varMatch[1]
			if val, ok := bindings[datalog.Variable(varName)]; ok {
				if numVal, err := toFloat64(val); err == nil {
					return datalog.FormatDuration(numVal)
				}
			}
		}
	}

	// Check if it's a simple variable reference
	if strings.HasPrefix(value, "?") {
		if val, ok := bindings[datalog.Variable(value)]; ok {
			return formatValue(val)
		}
	}

	// Otherwise render as template
	return renderTemplate(value, bindings)
}

// formatValue formats a value for display
func formatValue(val interface{}) string {
	switch v := val.(type) {
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.1f", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprint(v)
	}
}

// impactOrder returns sort order for impact levels
func impactOrder(impact string) int {
	switch impact {
	case "high":
		return 0
	case "medium":
		return 1
	case "low":
		return 2
	default:
		return 3
	}
}

// deduplicateSuggestions removes duplicate suggestions
func deduplicateSuggestions(suggestions []datalog.Suggestion) []datalog.Suggestion {
	seen := make(map[string]bool)
	var result []datalog.Suggestion

	for _, s := range suggestions {
		key := s.RuleID + ":" + s.Target
		if !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}

	return result
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
