// Package console implements ports/logger.Sink for an io.Writer
// destination — typically os.Stdout or os.Stderr.
//
// This is the reference adapter for the logger component and the
// simplest possible Sink: one EntryEvent → one line of output,
// synchronously serialised through a mutex.
//
// Implementation pending — see CLAUDE.md for the spec and
// components/logger/CLAUDE.md for the cahier des charges driving it.
package console
