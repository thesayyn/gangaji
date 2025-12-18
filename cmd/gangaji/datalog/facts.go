package datalog

// TraceEvent represents a trace event (mirrored from main package to avoid import cycle)
type TraceEvent struct {
	Name string                 `json:"name"`
	Cat  string                 `json:"cat,omitempty"`
	Ph   string                 `json:"ph"`
	Ts   float64                `json:"ts"`
	Dur  float64                `json:"dur,omitempty"`
	Pid  int                    `json:"pid,omitempty"`
	Tid  int                    `json:"tid,omitempty"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// isActionableCategory returns true if the category represents user-controlled work
func isActionableCategory(cat string) bool {
	switch cat {
	case "action processing", "complete action execution":
		return true
	case "Fetching repository": // External deps - actionable via MODULE.bazel
		return true
	case "package creation": // BUILD file loading
		return true
	default:
		return false
	}
}

// isSystemCategory returns true if the category represents Bazel infrastructure
func isSystemCategory(cat string) bool {
	switch cat {
	case "general information", "build phase marker", "gc notification",
		"skyframe evaluator", "action count (local)", "critical path component",
		"Conflict checking", "bazel module processing":
		return true
	default:
		return false
	}
}

// GenerateFacts generates Datalog facts from trace events
func GenerateFacts(events []TraceEvent) []Fact {
	facts := make([]Fact, 0, len(events)*4)

	var totalDuration float64
	var maxEnd float64
	var actionableTime float64
	var actionableCount int

	// First pass: compute totals and generate base facts
	for i, e := range events {
		// trace_event(id, name, category, start_us, duration_us)
		facts = append(facts, Fact{
			Predicate: "trace_event",
			Args:      []interface{}{i, e.Name, e.Cat, e.Ts, e.Dur},
		})

		// trace_event_tid(id, tid)
		facts = append(facts, Fact{
			Predicate: "trace_event_tid",
			Args:      []interface{}{i, e.Tid},
		})

		// trace_event_pid(id, pid)
		facts = append(facts, Fact{
			Predicate: "trace_event_pid",
			Args:      []interface{}{i, e.Pid},
		})

		// Extract mnemonic from args
		if mnemonic, ok := e.Args["mnemonic"].(string); ok {
			facts = append(facts, Fact{
				Predicate: "trace_event_mnemonic",
				Args:      []interface{}{i, mnemonic},
			})
		}

		// Extract target from args (Bazel label)
		if target, ok := e.Args["target"].(string); ok && target != "" {
			facts = append(facts, Fact{
				Predicate: "trace_event_target",
				Args:      []interface{}{i, target},
			})
			// Events with targets are user-controlled actions
			facts = append(facts, Fact{
				Predicate: "has_target",
				Args:      []interface{}{i},
			})
		}

		// Determine if event is actionable (user-controlled) vs system (Bazel infra)
		hasTarget := false
		if target, ok := e.Args["target"].(string); ok && target != "" {
			hasTarget = true
		}

		// An event is actionable if:
		// 1. It has a target label (user's BUILD files), OR
		// 2. It's in an actionable category AND has a mnemonic
		isActionable := hasTarget || (isActionableCategory(e.Cat) && e.Args["mnemonic"] != nil)

		if isActionable {
			facts = append(facts, Fact{
				Predicate: "is_actionable",
				Args:      []interface{}{i},
			})
			actionableTime += e.Dur
			actionableCount++
		}

		if isSystemCategory(e.Cat) {
			facts = append(facts, Fact{
				Predicate: "is_system",
				Args:      []interface{}{i},
			})
		}

		// Track total duration and max end time
		end := e.Ts + e.Dur
		if end > maxEnd {
			maxEnd = end
		}
		totalDuration += e.Dur
	}

	// Add aggregate facts
	facts = append(facts, Fact{
		Predicate: "total_duration",
		Args:      []interface{}{maxEnd},
	})

	facts = append(facts, Fact{
		Predicate: "total_action_time",
		Args:      []interface{}{totalDuration},
	})

	facts = append(facts, Fact{
		Predicate: "total_actions",
		Args:      []interface{}{len(events)},
	})

	// Add actionable aggregate facts (user-controlled work)
	facts = append(facts, Fact{
		Predicate: "actionable_time",
		Args:      []interface{}{actionableTime},
	})

	facts = append(facts, Fact{
		Predicate: "actionable_count",
		Args:      []interface{}{actionableCount},
	})

	// Compute category aggregates
	categoryTime := make(map[string]float64)
	categoryCount := make(map[string]int)
	for _, e := range events {
		categoryTime[e.Cat] += e.Dur
		categoryCount[e.Cat]++
	}

	for cat, time := range categoryTime {
		facts = append(facts, Fact{
			Predicate: "category_time",
			Args:      []interface{}{cat, time},
		})
	}

	for cat, count := range categoryCount {
		facts = append(facts, Fact{
			Predicate: "category_count",
			Args:      []interface{}{cat, count},
		})
	}

	// Compute mnemonic aggregates (only for actionable events with targets)
	mnemonicTime := make(map[string]float64)
	mnemonicCount := make(map[string]int)
	for _, e := range events {
		if mnemonic, ok := e.Args["mnemonic"].(string); ok {
			// Only count if has target (user-controlled action)
			if target, ok := e.Args["target"].(string); ok && target != "" {
				mnemonicTime[mnemonic] += e.Dur
				mnemonicCount[mnemonic]++
			}
		}
	}

	for mnemonic, time := range mnemonicTime {
		facts = append(facts, Fact{
			Predicate: "mnemonic_time",
			Args:      []interface{}{mnemonic, time},
		})
	}

	for mnemonic, count := range mnemonicCount {
		facts = append(facts, Fact{
			Predicate: "mnemonic_count",
			Args:      []interface{}{mnemonic, count},
		})
	}

	// Compute target-based aggregates (by Bazel package)
	targetTime := make(map[string]float64)
	targetCount := make(map[string]int)
	for _, e := range events {
		if target, ok := e.Args["target"].(string); ok && target != "" {
			targetTime[target] += e.Dur
			targetCount[target]++
		}
	}

	// Add facts for slow targets (user's actual build targets)
	for target, time := range targetTime {
		facts = append(facts, Fact{
			Predicate: "target_time",
			Args:      []interface{}{target, time},
		})
	}

	// Compute concurrency (max overlapping events)
	maxConcurrency := computeMaxConcurrency(events)
	facts = append(facts, Fact{
		Predicate: "max_concurrency",
		Args:      []interface{}{maxConcurrency},
	})

	// Compute critical path info
	criticalPathFacts := computeCriticalPath(events)
	facts = append(facts, criticalPathFacts...)

	return facts
}

// computeMaxConcurrency computes the maximum number of concurrent events
func computeMaxConcurrency(events []TraceEvent) int {
	if len(events) == 0 {
		return 0
	}

	// Create a list of start and end times
	type timePoint struct {
		time    float64
		isStart bool
	}

	points := make([]timePoint, 0, len(events)*2)
	for _, e := range events {
		points = append(points, timePoint{e.Ts, true})
		points = append(points, timePoint{e.Ts + e.Dur, false})
	}

	// Sort by time, with starts before ends at the same time
	for i := 0; i < len(points)-1; i++ {
		for j := i + 1; j < len(points); j++ {
			if points[i].time > points[j].time ||
				(points[i].time == points[j].time && !points[i].isStart && points[j].isStart) {
				points[i], points[j] = points[j], points[i]
			}
		}
	}

	// Sweep through and track max concurrent
	maxConcurrent := 0
	current := 0
	for _, p := range points {
		if p.isStart {
			current++
			if current > maxConcurrent {
				maxConcurrent = current
			}
		} else {
			current--
		}
	}

	return maxConcurrent
}

// computeCriticalPath identifies events on the critical path
func computeCriticalPath(events []TraceEvent) []Fact {
	if len(events) == 0 {
		return nil
	}

	var facts []Fact

	// Find the max end time for all events
	var maxEnd float64
	for _, e := range events {
		end := e.Ts + e.Dur
		if end > maxEnd {
			maxEnd = end
		}
	}

	// Find the actionable event that ends last (critical path endpoint)
	var lastActionableEvent *TraceEvent
	var lastActionableEventIdx int
	var lastActionableEnd float64

	for i, e := range events {
		// Only consider actionable events (those with targets)
		if target, ok := e.Args["target"].(string); ok && target != "" {
			end := e.Ts + e.Dur
			if end > lastActionableEnd {
				lastActionableEnd = end
				lastActionableEvent = &events[i]
				lastActionableEventIdx = i
			}
		}
	}

	if lastActionableEvent != nil {
		target := ""
		if t, ok := lastActionableEvent.Args["target"].(string); ok {
			target = t
		}
		// Mark as critical path endpoint
		facts = append(facts, Fact{
			Predicate: "critical_path_end",
			Args:      []interface{}{lastActionableEventIdx, lastActionableEvent.Name, lastActionableEvent.Dur, target},
		})

		// Calculate critical path percentage
		if maxEnd > 0 {
			criticalPathPct := (lastActionableEvent.Dur / maxEnd) * 100
			facts = append(facts, Fact{
				Predicate: "critical_path_percent",
				Args:      []interface{}{criticalPathPct},
			})
		}
	}

	// Find top bottlenecks among actionable events only
	type actionableEvent struct {
		idx      int
		duration float64
		name     string
		target   string
	}
	var actionableEvents []actionableEvent

	for i, e := range events {
		// Only consider actionable events (those with targets)
		if target, ok := e.Args["target"].(string); ok && target != "" {
			actionableEvents = append(actionableEvents, actionableEvent{
				idx:      i,
				duration: e.Dur,
				name:     e.Name,
				target:   target,
			})
		}
	}

	// Sort by duration descending
	for i := 0; i < len(actionableEvents)-1; i++ {
		for j := i + 1; j < len(actionableEvents); j++ {
			if actionableEvents[i].duration < actionableEvents[j].duration {
				actionableEvents[i], actionableEvents[j] = actionableEvents[j], actionableEvents[i]
			}
		}
	}

	// Mark top 5 actionable events as potential bottlenecks
	for i := 0; i < 5 && i < len(actionableEvents); i++ {
		e := actionableEvents[i]
		pct := float64(0)
		if maxEnd > 0 {
			pct = (e.duration / maxEnd) * 100
		}
		facts = append(facts, Fact{
			Predicate: "potential_bottleneck",
			Args:      []interface{}{e.idx, e.name, e.duration, pct, e.target},
		})
	}

	return facts
}

// GenerateEventPercentFacts generates event percentage facts
func GenerateEventPercentFacts(events []TraceEvent, totalDuration float64) []Fact {
	var facts []Fact

	for i, e := range events {
		if totalDuration > 0 {
			pct := (e.Dur / totalDuration) * 100
			facts = append(facts, Fact{
				Predicate: "event_percent",
				Args:      []interface{}{i, pct},
			})
		}
	}

	return facts
}
