package errs

// ResetForTest clears the error registry and resets the ID counter.
// Exported for use in external (_test) packages only.
func ResetForTest() {
	clearRegistry()
}
