package build_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/michaelbomholt665/go-tree-sitter/internal/filelock"
	"github.com/michaelbomholt665/go-tree-sitter/internal/tests/testutil"
	ts "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter"
	build "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func TestReviewMoveRequiresForceAndPreservesCatalog(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("atomic directory exchange unavailable")
	}
	cfg, _, _ := policyWorkspace(t)
	mover := build.NewMoverWithValidator(nil, acceptingValidator{}, nil, nil)
	req := ts.MoveRequest{Mode: ts.MoveModeBoth}
	if err := mover.Move(context.Background(), cfg, req); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(cfg.Output.GrammarBase, "python", "old-marker")
	policyWrite(t, marker, "preserve")
	if err := mover.Move(context.Background(), cfg, req); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected force requirement: %v", err)
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "preserve" {
		t.Fatalf("catalog changed: %q %v", raw, err)
	}
	req.Force = true
	if err := mover.Move(context.Background(), cfg, req); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("forced replacement retained stale file: %v", err)
	}
}

func TestReviewMoveRejectsChangedBinary(t *testing.T) {
	cfg, _, binary := policyWorkspace(t)
	policyWrite(t, binary, "changed after compilation")
	mover := build.NewMoverWithValidator(nil, acceptingValidator{}, nil, nil)
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{Mode: ts.MoveModeBoth}); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("accepted changed artifact: %v", err)
	}
	if _, err := os.Stat(cfg.Output.GrammarBase); !os.IsNotExist(err) {
		t.Fatalf("published invalid artifact: %v", err)
	}
}

func TestReviewMoveLockPrecedesStaging(t *testing.T) {
	cfg, _, _ := policyWorkspace(t)
	release, err := filelock.Lock(cfg.Output.GrammarBase + ".lock")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	mover := build.NewMoverWithValidator(nil, acceptingValidator{}, nil, nil)
	if err := mover.Move(context.Background(), cfg, ts.MoveRequest{}); err == nil || !strings.Contains(err.Error(), "lock") {
		t.Fatalf("ignored publisher lock: %v", err)
	}
	if _, err := os.Stat(cfg.Output.GrammarBase); !os.IsNotExist(err) {
		t.Fatalf("changed locked catalog: %v", err)
	}
}

type cancelValidation struct{ cancel context.CancelFunc }

func (v cancelValidation) Validate(ctx context.Context, items []build.ValidationItem) (map[string]uint32, error) {
	result, err := (acceptingValidator{}).Validate(ctx, items)
	v.cancel()
	return result, err
}
func TestReviewCancellationAfterValidationPreventsPublication(t *testing.T) {
	cfg, source, _ := policyWorkspace(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mover := build.NewMoverWithValidator(nil, cancelValidation{cancel}, nil, nil)
	err := mover.Move(ctx, cfg, ts.MoveRequest{Mode: ts.MoveModeBoth, Clean: true})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	if _, err := os.Stat(cfg.Output.GrammarBase); !os.IsNotExist(err) {
		t.Fatalf("published after cancellation: %v", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("cleaned after cancellation: %v", err)
	}
}

func TestReviewBuilderRejectsUnpinnedRevisionWithoutRemovingWorkspace(t *testing.T) {
	cfg, source, _ := policyWorkspace(t)
	cfg.Languages[0].Revision = "main"
	builder := build.NewBuilder(&testutil.RecordingRunner{}, nil, nil)
	if err := builder.Build(context.Background(), cfg, ts.BuildRequest{Force: true}); err == nil {
		t.Fatal("accepted unpinned revision")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("removed workspace before validating pins: %v", err)
	}
}
