package main

import (
	"compress/gzip"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
	"github.com/thesayyn/gangaji/cmd/gangaji/datalog"
	"github.com/thesayyn/gangaji/cmd/gangaji/suggestions"
)

//go:embed flamegraph.html
var flamegraphHTML embed.FS

// TraceEvent represents a single event in Chrome Trace Event Format
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

// ThreadMetadata holds pre-processed thread information
type ThreadMetadata struct {
	Name      string `json:"name"`
	SortIndex *int   `json:"sortIndex,omitempty"`
}

// CounterEvent represents a counter event (ph: "C") for metrics like memory usage
type CounterEvent struct {
	Name string                 `json:"name"`
	Ts   float64                `json:"ts"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// ProfileData represents the complete profile data structure
type ProfileData struct {
	TraceEvents    []TraceEvent               `json:"traceEvents"`
	CounterEvents  []CounterEvent             `json:"counterEvents,omitempty"`
	ThreadMetadata map[int]*ThreadMetadata    `json:"threadMetadata,omitempty"`
	MainThreadTid  *int                       `json:"mainThreadTid,omitempty"`
}

var (
	profilePath         string
	starlarkProfilePath string
	rulesDir            string
	port                int
	openBrowserFlag     bool
)

func init() {
	flag.StringVar(&profilePath, "profile", "", "Path to Bazel profile JSON (can be .json or .json.gz)")
	flag.StringVar(&starlarkProfilePath, "starlark_cpu_profile", "", "Path to Starlark CPU profile")
	flag.StringVar(&rulesDir, "rules_dir", "", "Path to directory with custom .dl rule files (optional)")
	flag.IntVar(&port, "port", 8080, "HTTP server port")
	flag.BoolVar(&openBrowserFlag, "open", true, "Open browser automatically")
}

func main() {
	flag.Parse()

	if profilePath == "" && starlarkProfilePath == "" {
		fmt.Println("Gangaji - Bazel Build Profiler")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  gangaji --profile=<path> [--starlark_cpu_profile=<path>] [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Build profile only")
		fmt.Println("  gangaji --profile=profile.json")
		fmt.Println()
		fmt.Println("  # Starlark CPU profile only")
		fmt.Println("  gangaji --starlark_cpu_profile=starlark.json")
		fmt.Println()
		fmt.Println("  # Both profiles combined")
		fmt.Println("  gangaji --profile=profile.json --starlark_cpu_profile=starlark.json")
		os.Exit(1)
	}

	// Load profile data
	profileData, err := loadProfiles()
	if err != nil {
		log.Fatalf("Error loading profile: %v", err)
	}

	// Print what was loaded
	var sources []string
	if profilePath != "" {
		sources = append(sources, fmt.Sprintf("build profile (%s)", profilePath))
	}
	if starlarkProfilePath != "" {
		sources = append(sources, fmt.Sprintf("starlark profile (%s)", starlarkProfilePath))
	}
	fmt.Printf("Loaded %d trace events from %s\n", len(profileData.TraceEvents), strings.Join(sources, " + "))

	// Convert trace events for Datalog evaluation
	datalogEvents := convertToDatalogEvents(profileData.TraceEvents)

	// Initialize and run suggestions evaluator
	evaluator := suggestions.NewEvaluator(rulesDir)
	if err := evaluator.LoadRules(); err != nil {
		log.Printf("Warning: Failed to load rules: %v", err)
	}

	suggestionsResult, err := evaluator.Evaluate(datalogEvents)
	if err != nil {
		log.Printf("Warning: Failed to evaluate rules: %v", err)
		suggestionsResult = &suggestions.SuggestionsResult{}
	}

	fmt.Printf("Generated %d suggestions from %d rules\n", len(suggestionsResult.Suggestions), suggestionsResult.RulesEvaluated)

	// Create HTTP server
	server := &Server{
		profileData:       profileData,
		suggestionsResult: suggestionsResult,
	}

	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/api/profile", server.handleProfileAPI)
	http.HandleFunc("/api/suggestions", server.handleSuggestionsAPI)

	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	fmt.Printf("Starting Gangaji server at %s\n", url)
	fmt.Println("Press Ctrl+C to stop")

	if openBrowserFlag {
		go openBrowser(url)
	}

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadProfiles() (*ProfileData, error) {
	result := &ProfileData{
		TraceEvents:    []TraceEvent{},
		CounterEvents:  []CounterEvent{},
		ThreadMetadata: make(map[int]*ThreadMetadata),
	}

	// Load Bazel profile
	if profilePath != "" {
		data, err := loadBazelProfile(profilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load Bazel profile: %w", err)
		}
		result.TraceEvents = append(result.TraceEvents, data.TraceEvents...)
		result.CounterEvents = append(result.CounterEvents, data.CounterEvents...)

		// Copy thread metadata
		for tid, meta := range data.ThreadMetadata {
			result.ThreadMetadata[tid] = meta
		}
		if data.MainThreadTid != nil {
			result.MainThreadTid = data.MainThreadTid
		}
	}

	// Load Starlark CPU profile
	if starlarkProfilePath != "" {
		events, err := loadStarlarkProfile(starlarkProfilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load Starlark profile: %w", err)
		}
		result.TraceEvents = append(result.TraceEvents, events...)
	}

	return result, nil
}

func loadBazelProfile(path string) (*ProfileData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var reader io.Reader = file

	// Check if gzipped
	if strings.HasSuffix(path, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	// Parse JSON
	var profile ProfileData
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile JSON: %w", err)
	}

	// Extract thread metadata from metadata events
	profile.ThreadMetadata = make(map[int]*ThreadMetadata)

	for _, event := range profile.TraceEvents {
		// Process thread_name metadata events
		if event.Name == "thread_name" && event.Ph == "M" {
			tid := event.Tid
			if profile.ThreadMetadata[tid] == nil {
				profile.ThreadMetadata[tid] = &ThreadMetadata{}
			}
			if name, ok := event.Args["name"].(string); ok {
				profile.ThreadMetadata[tid].Name = name
				// Detect main thread
				if name == "Main Thread" {
					profile.MainThreadTid = &tid
				}
			}
		}

		// Process thread_sort_index metadata events
		if event.Name == "thread_sort_index" && event.Ph == "M" {
			tid := event.Tid
			if profile.ThreadMetadata[tid] == nil {
				profile.ThreadMetadata[tid] = &ThreadMetadata{}
			}
			if sortIndex, ok := event.Args["sort_index"].(float64); ok {
				idx := int(sortIndex)
				profile.ThreadMetadata[tid].SortIndex = &idx
			}
		}
	}

	// Filter events: complete events (ph: "X") go to TraceEvents, counter events (ph: "C") go to CounterEvents
	filtered := make([]TraceEvent, 0, len(profile.TraceEvents))
	counterEvents := make([]CounterEvent, 0)
	for _, event := range profile.TraceEvents {
		if event.Ph == "X" && event.Dur > 0 {
			filtered = append(filtered, event)
		} else if event.Ph == "C" {
			counterEvents = append(counterEvents, CounterEvent{
				Name: event.Name,
				Ts:   event.Ts,
				Args: event.Args,
			})
		}
	}
	profile.TraceEvents = filtered
	profile.CounterEvents = counterEvents

	return &profile, nil
}

func loadStarlarkProfile(path string) ([]TraceEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Try parsing as pprof format
	prof, err := profile.Parse(file)
	if err != nil {
		// Try JSON trace format as fallback
		file.Seek(0, 0)
		return loadStarlarkProfileJSON(file)
	}

	return convertPprofToTraceEvents(prof), nil
}

func loadStarlarkProfileJSON(reader io.Reader) ([]TraceEvent, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var profileData ProfileData
	if err := json.Unmarshal(data, &profileData); err != nil {
		log.Printf("Warning: Starlark profile format not recognized, skipping")
		return []TraceEvent{}, nil
	}

	events := make([]TraceEvent, 0)
	for _, event := range profileData.TraceEvents {
		if event.Ph == "X" && event.Dur > 0 {
			event.Cat = "starlark"
			events = append(events, event)
		}
	}
	return events, nil
}

func convertPprofToTraceEvents(prof *profile.Profile) []TraceEvent {
	events := make([]TraceEvent, 0)

	// Get the time unit (usually nanoseconds or microseconds)
	timeUnit := float64(1) // default to 1 (nanoseconds)
	for _, st := range prof.SampleType {
		if st.Type == "cpu" || st.Type == "samples" {
			if st.Unit == "microseconds" {
				timeUnit = 1000 // convert to nanoseconds for consistency
			} else if st.Unit == "nanoseconds" {
				timeUnit = 1
			} else if st.Unit == "milliseconds" {
				timeUnit = 1000000
			}
		}
	}

	// Aggregate samples by function to create synthetic trace events
	functionTimes := make(map[string]int64)
	functionFiles := make(map[string]string)

	for _, sample := range prof.Sample {
		if len(sample.Location) == 0 {
			continue
		}

		// Get the value (CPU time)
		value := int64(0)
		if len(sample.Value) > 0 {
			value = sample.Value[0]
		}

		// Walk the call stack
		for _, loc := range sample.Location {
			for _, line := range loc.Line {
				if line.Function != nil {
					funcName := line.Function.Name
					functionTimes[funcName] += value
					if line.Function.Filename != "" {
						functionFiles[funcName] = line.Function.Filename
					}
				}
			}
		}
	}

	// Sort functions by time (descending)
	type funcTime struct {
		name string
		time int64
	}
	var sorted []funcTime
	for name, time := range functionTimes {
		sorted = append(sorted, funcTime{name, time})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].time > sorted[j].time
	})

	// Create trace events with synthetic timestamps
	// Lay them out sequentially for visualization
	currentTs := float64(0)
	for _, ft := range sorted {
		if ft.time <= 0 {
			continue
		}

		// Duration in microseconds (what the flamegraph expects)
		durUs := float64(ft.time) * timeUnit / 1000

		event := TraceEvent{
			Name: ft.name,
			Cat:  "starlark",
			Ph:   "X",
			Ts:   currentTs,
			Dur:  durUs,
			Args: map[string]interface{}{
				"mnemonic": "Starlark",
			},
		}

		if file, ok := functionFiles[ft.name]; ok {
			event.Args["file"] = file
		}

		events = append(events, event)
		currentTs += durUs
	}

	return events
}

// Server handles HTTP requests
type Server struct {
	profileData       *ProfileData
	suggestionsResult *suggestions.SuggestionsResult
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Generate HTML with embedded profile data
	profileJSON, err := json.Marshal(s.profileData)
	if err != nil {
		http.Error(w, "Failed to serialize profile data", http.StatusInternalServerError)
		return
	}

	html := generateHTML(string(profileJSON))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *Server) handleProfileAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(s.profileData)
}

func (s *Server) handleSuggestionsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(s.suggestionsResult)
}

func generateHTML(profileJSON string) string {
	// Read embedded flamegraph HTML
	htmlBytes, err := flamegraphHTML.ReadFile("flamegraph.html")
	if err != nil {
		log.Fatalf("Failed to read embedded flamegraph.html: %v", err)
	}

	html := string(htmlBytes)

	// Inject profile data - replace the DEMO_DATA or add before </head>
	dataScript := fmt.Sprintf(`<script>window.BAZEL_PROFILE_DATA = %s;</script>`, profileJSON)

	// Try to find and replace demo data placeholder
	if strings.Contains(html, "const DEMO_DATA") {
		// Find the start of DEMO_DATA and replace until the closing brace
		startIdx := strings.Index(html, "const DEMO_DATA")
		if startIdx != -1 {
			// Find the line end
			endIdx := strings.Index(html[startIdx:], "};")
			if endIdx != -1 {
				endIdx += startIdx + 2
				html = html[:startIdx] + "const DEMO_DATA = window.BAZEL_PROFILE_DATA || { traceEvents: [] }" + html[endIdx:]
			}
		}
	}

	// Insert data script before closing </head> tag
	if idx := strings.Index(html, "</head>"); idx != -1 {
		html = html[:idx] + dataScript + "\n" + html[idx:]
	} else if idx := strings.Index(html, "<body"); idx != -1 {
		html = html[:idx] + dataScript + "\n" + html[idx:]
	}

	return html
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

// convertToDatalogEvents converts main.TraceEvent to datalog.TraceEvent
func convertToDatalogEvents(events []TraceEvent) []datalog.TraceEvent {
	result := make([]datalog.TraceEvent, len(events))
	for i, e := range events {
		result[i] = datalog.TraceEvent{
			Name: e.Name,
			Cat:  e.Cat,
			Ph:   e.Ph,
			Ts:   e.Ts,
			Dur:  e.Dur,
			Pid:  e.Pid,
			Tid:  e.Tid,
			Args: e.Args,
		}
	}
	return result
}
