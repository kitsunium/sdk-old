package normalize

// Value normalizes the input string by trimming surrounding spaces
// and removing surrounding single or double quotes if present.
//
// Parameters:
// - value: string - The input string to normalize.
//
// Returns:
// - string: The normalized string with spaces and quotes removed if applicable.
func Value(value string) string {
	// Trim spaces manually without using strings.TrimSpace
	start, end := 0, len(value)

	for start < end && (value[start] == ' ' || value[start] == '\t' || value[start] == '\n') {
		start++
	}
	for start < end && (value[end-1] == ' ' || value[end-1] == '\t' || value[end-1] == '\n') {
		end--
	}

	// Check for surrounding quotes if at least two characters remain
	if end-start >= 2 {
		if value[start] == '\'' && value[end-1] == '\'' {
			start++
			end--
		} else if value[start] == '"' && value[end-1] == '"' {
			start++
			end--
		}
	}

	// Return the sliced and normalized string
	return value[start:end]
}
