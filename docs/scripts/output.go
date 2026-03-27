package main

import (
	"fmt"
	"io"
	"os"
)

func mustFprintf(w io.Writer, format string, args ...any) {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		panic(fmt.Errorf("write output: %w", err))
	}
}

func mustFprintln(w io.Writer, args ...any) {
	if _, err := fmt.Fprintln(w, args...); err != nil {
		panic(fmt.Errorf("write output: %w", err))
	}
}

func mustWriteString(w io.StringWriter, text string) {
	if _, err := w.WriteString(text); err != nil {
		panic(fmt.Errorf("write output: %w", err))
	}
}

func stdoutf(format string, args ...any) {
	mustFprintf(os.Stdout, format, args...)
}

func stdoutln(args ...any) {
	mustFprintln(os.Stdout, args...)
}

func stderrf(format string, args ...any) {
	mustFprintf(os.Stderr, format, args...)
}
