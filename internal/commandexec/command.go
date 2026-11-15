// Package commandexec runs commands with descendant cancellation.
package commandexec

import (
	"context"
	"os/exec"
	"time"
)

func New(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	name, args, err := resolveCommand(name, args)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = 2 * time.Second
	configure(cmd)
	return cmd, nil
}
