package treesitter

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestReviewCommandOutputHelper(t *testing.T) {
	if os.Getenv("TS_REVIEW_OUTPUT_HELPER") != "1" {
		return
	}
	_, _ = os.Stdout.WriteString(strings.Repeat("x", 2<<20) + "diagnostic-tail")
	os.Exit(1)
}

func TestReviewRunnerBoundsOutput(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	runner := NewVerboseExecRunner(io.Discard, io.Discard, false)
	cmd := Command{Name: exe, Args: []string{"-test.run=^TestReviewCommandOutputHelper$"}, Env: map[string]string{"TS_REVIEW_OUTPUT_HELPER": "1"}}
	err = runner.Run(context.Background(), cmd)
	if err == nil || !strings.Contains(err.Error(), "diagnostic-tail") || len(err.Error()) > 70<<10 {
		t.Fatalf("unexpected diagnostic size/tail: %v", err)
	}
	_, err = runner.Output(context.Background(), cmd)
	if err == nil || !strings.Contains(err.Error(), "output exceeded") {
		t.Fatalf("expected output limit error: %v", err)
	}
}

func TestReviewTailBuffer(t *testing.T) {
	b := newTailBuffer(4)
	for _, text := range []string{"ab", "cde", "012345"} {
		_, _ = b.Write([]byte(text))
	}
	if b.String() != "2345" || !b.overflow {
		t.Fatalf("tail=%q overflow=%v", b.String(), b.overflow)
	}
}

func TestReviewRejectsDangerousPositionalArguments(t *testing.T) {
	for _, command := range []string{"clean", "build", "compile", "move"} {
		t.Run(command, func(t *testing.T) {
			app := NewApp(io.Discard, io.Discard, nil, nil, nil, nil, nil)
			err := app.Run(context.Background(), []string{command, "python"})
			if err == nil {
				t.Fatal("expected positional argument rejection")
			}
		})
	}
}
