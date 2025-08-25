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
	start, end := trimWhitespace(value)
	start, end = trimQuotes(value, start, end)
	return value[start:end]
}

func trimWhitespace(value string) (int, int) {
	start, end := 0, len(value)

	for start < end && isWhitespace(value[start]) {
		start++
	}
	for start < end && isWhitespace(value[end-1]) {
		end--
	}

	return start, end
}

func trimQuotes(value string, start, end int) (int, int) {
	if end-start < 2 {
		return start, end
	}

	if (value[start] == '\'' && value[end-1] == '\'') ||
		(value[start] == '"' && value[end-1] == '"') {
		return start + 1, end - 1
	}

	return start, end
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}
