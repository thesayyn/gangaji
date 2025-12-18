package datalog

import (
	"fmt"
	"math"
	"strings"
)

// RegisterFormattingBuiltins registers formatting built-in functions
func (e *Engine) RegisterFormattingBuiltins() {
	// format_time formats microseconds to human-readable string
	e.RegisterBuiltin("format_time", func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("format_time expects 1 argument")
		}
		us, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		return FormatDuration(us), nil
	})

	// format_percent formats a number as percentage
	e.RegisterBuiltin("format_percent", func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("format_percent expects 1 argument")
		}
		val, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("%.1f%%", val), nil
	})

	// format_number formats a number with commas
	e.RegisterBuiltin("format_number", func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("format_number expects 1 argument")
		}
		val, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		return formatWithCommas(int64(val)), nil
	})

	// round_to rounds to n decimal places
	e.RegisterBuiltin("round_to", func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("round_to expects 2 arguments")
		}
		val, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		places, err := toFloat64(args[1])
		if err != nil {
			return nil, err
		}
		mult := math.Pow(10, places)
		return math.Round(val*mult) / mult, nil
	})

	// truncate truncates a string to max length
	e.RegisterBuiltin("truncate", func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("truncate expects 2 arguments")
		}
		str := fmt.Sprint(args[0])
		maxLen, err := toFloat64(args[1])
		if err != nil {
			return nil, err
		}
		if len(str) > int(maxLen) {
			return str[:int(maxLen)-3] + "...", nil
		}
		return str, nil
	})

	// concat concatenates strings
	e.RegisterBuiltin("concat", func(args []interface{}) (interface{}, error) {
		var parts []string
		for _, arg := range args {
			parts = append(parts, fmt.Sprint(arg))
		}
		return strings.Join(parts, ""), nil
	})

	// contains checks if string contains substring
	e.RegisterBuiltin("contains", func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("contains expects 2 arguments")
		}
		str := fmt.Sprint(args[0])
		substr := fmt.Sprint(args[1])
		return strings.Contains(str, substr), nil
	})

	// starts_with checks if string starts with prefix
	e.RegisterBuiltin("starts_with", func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("starts_with expects 2 arguments")
		}
		str := fmt.Sprint(args[0])
		prefix := fmt.Sprint(args[1])
		return strings.HasPrefix(str, prefix), nil
	})

	// ends_with checks if string ends with suffix
	e.RegisterBuiltin("ends_with", func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("ends_with expects 2 arguments")
		}
		str := fmt.Sprint(args[0])
		suffix := fmt.Sprint(args[1])
		return strings.HasSuffix(str, suffix), nil
	})

	// min returns the minimum of two values
	e.RegisterBuiltin("min", func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("min expects 2 arguments")
		}
		a, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		b, err := toFloat64(args[1])
		if err != nil {
			return nil, err
		}
		if a < b {
			return a, nil
		}
		return b, nil
	})

	// max returns the maximum of two values
	e.RegisterBuiltin("max", func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("max expects 2 arguments")
		}
		a, err := toFloat64(args[0])
		if err != nil {
			return nil, err
		}
		b, err := toFloat64(args[1])
		if err != nil {
			return nil, err
		}
		if a > b {
			return a, nil
		}
		return b, nil
	})
}

// FormatDuration formats microseconds to human-readable duration
func FormatDuration(us float64) string {
	if us < 1000 {
		return fmt.Sprintf("%.0fμs", us)
	}
	ms := us / 1000
	if ms < 1000 {
		return fmt.Sprintf("%.1fms", ms)
	}
	s := ms / 1000
	if s < 60 {
		return fmt.Sprintf("%.2fs", s)
	}
	m := s / 60
	if m < 60 {
		remainingS := int(s) % 60
		return fmt.Sprintf("%.0fm %ds", m, remainingS)
	}
	h := m / 60
	remainingM := int(m) % 60
	return fmt.Sprintf("%.0fh %dm", h, remainingM)
}

// formatWithCommas formats a number with thousands separators
func formatWithCommas(n int64) string {
	str := fmt.Sprintf("%d", n)
	if n < 0 {
		str = str[1:] // Remove negative sign temporarily
	}

	// Insert commas
	var result []byte
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}

	if n < 0 {
		return "-" + string(result)
	}
	return string(result)
}
