package treesitter

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Reporter is the interface for reporting build progress.
// Implementations must be safe for concurrent use.
type Reporter interface {
	// Start optionally logs that work on step/target has begun (used for
	// verbose streaming; silent in clean mode).
	Start(step, target string)

	// Success prints a single clean status line when a step finishes
	// successfully.
	Success(step, target, detail string, elapsed time.Duration)

	// Failure prints a failure banner and the captured diagnostic output.
	Failure(step, target string, err error, diagnostics string)

	// Info prints an informational message (e.g. a summary line).
	Info(format string, args ...any)
}

// ─── TextReporter ────────────────────────────────────────────────────────────

// TextReporter prints human-readable progress to the supplied writer.  When
// the writer is a terminal it uses ANSI colour; when piped it produces plain
// text suitable for CI logs.
type TextReporter struct {
	w       io.Writer
	verbose bool
	color   bool
}

// NewTextReporter returns a Reporter that writes to w.  Pass verbose=true to
// enable streaming mode (equivalent to --verbose).
func NewTextReporter(w io.Writer, verbose bool) *TextReporter {
	color := false
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			color = true
		}
	}
	return &TextReporter{w: w, verbose: verbose, color: color}
}

func (r *TextReporter) Start(step, target string) {
	if !r.verbose {
		return
	}
	fmt.Fprintf(r.w, "[ ] %-9s %s\n", step, target)
}

func (r *TextReporter) Success(step, target, detail string, elapsed time.Duration) {
	marker := r.green("[✓]")
	detail = strings.TrimSpace(detail)
	if detail != "" {
		fmt.Fprintf(r.w, "%s %-9s %-30s (%s)\n", marker, step, target+" ["+detail+"]", fmtDuration(elapsed))
	} else {
		fmt.Fprintf(r.w, "%s %-9s %-30s (%s)\n", marker, step, target, fmtDuration(elapsed))
	}
}

func (r *TextReporter) Failure(step, target string, err error, diagnostics string) {
	marker := r.red("[✗]")
	fmt.Fprintf(r.w, "%s %-9s %-30s (FAILED)\n", marker, step, target)
	if diagnostics != "" {
		sep := strings.Repeat("─", 62)
		fmt.Fprintf(r.w, "── Error Diagnostics %s\n", sep[:len(sep)-len("── Error Diagnostics ")])
		fmt.Fprintln(r.w, strings.TrimSpace(diagnostics))
		fmt.Fprintln(r.w, sep)
	} else if err != nil {
		sep := strings.Repeat("─", 62)
		fmt.Fprintf(r.w, "── Error Diagnostics %s\n", sep[:len(sep)-len("── Error Diagnostics ")])
		fmt.Fprintln(r.w, err.Error())
		fmt.Fprintln(r.w, sep)
	}
}

func (r *TextReporter) Info(format string, args ...any) {
	fmt.Fprintf(r.w, format+"\n", args...)
}

func (r *TextReporter) green(s string) string {
	if r.color {
		return "\033[32m" + s + "\033[0m"
	}
	return s
}

func (r *TextReporter) red(s string) string {
	if r.color {
		return "\033[31m" + s + "\033[0m"
	}
	return s
}

func fmtDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// ─── SilentReporter ──────────────────────────────────────────────────────────

// SilentReporter discards all output.  Useful for tests and --quiet mode.
type SilentReporter struct{}

func (SilentReporter) Start(_, _ string)                       {}
func (SilentReporter) Success(_, _, _ string, _ time.Duration) {}
func (SilentReporter) Failure(_, _ string, _ error, _ string)  {}
func (SilentReporter) Info(_ string, _ ...any)                 {}
