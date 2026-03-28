package cobras

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

type fakeOpts struct {
	completeCalled bool
	validateCalled bool
	runCalled      bool
	completeErr    error
	validateErr    error
	runErr         error
}

func (f *fakeOpts) Complete(cmd *cobra.Command, args []string) error {
	f.completeCalled = true
	return f.completeErr
}

func (f *fakeOpts) Validate() error {
	f.validateCalled = true
	return f.validateErr
}

func (f *fakeOpts) Run(ctx context.Context) error {
	f.runCalled = true
	return f.runErr
}

func TestRun(t *testing.T) {
	opts := &fakeOpts{}
	cmd := &cobra.Command{
		Run: Run(opts),
	}

	if err := ExecuteE(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opts.completeCalled {
		t.Fatal("expected Complete to be called")
	}
	if !opts.validateCalled {
		t.Fatal("expected Validate to be called")
	}
	if !opts.runCalled {
		t.Fatal("expected Run to be called")
	}
}

func TestRunE(t *testing.T) {
	opts := &fakeOpts{}
	cmd := &cobra.Command{
		RunE: RunE(opts),
	}

	if err := ExecuteE(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opts.completeCalled {
		t.Fatal("expected Complete to be called")
	}
	if !opts.validateCalled {
		t.Fatal("expected Validate to be called")
	}
	if !opts.runCalled {
		t.Fatal("expected Run to be called")
	}
}

func TestRunECompleteError(t *testing.T) {
	opts := &fakeOpts{completeErr: errors.New("complete failed")}
	cmd := &cobra.Command{
		RunE:          RunE(opts),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	if err := ExecuteE(cmd); err == nil {
		t.Fatal("expected error from Complete")
	}
	if opts.validateCalled {
		t.Fatal("Validate should not be called when Complete fails")
	}
	if opts.runCalled {
		t.Fatal("Run should not be called when Complete fails")
	}
}

func TestRunEValidateError(t *testing.T) {
	opts := &fakeOpts{validateErr: errors.New("validate failed")}
	cmd := &cobra.Command{
		RunE:          RunE(opts),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	if err := ExecuteE(cmd); err == nil {
		t.Fatal("expected error from Validate")
	}
	if opts.runCalled {
		t.Fatal("Run should not be called when Validate fails")
	}
}

func TestRunERunError(t *testing.T) {
	opts := &fakeOpts{runErr: errors.New("run failed")}
	cmd := &cobra.Command{
		RunE:          RunE(opts),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	err := ExecuteE(cmd)
	if err == nil || err.Error() != "run failed" {
		t.Fatalf("expected 'run failed', got %v", err)
	}
}

func TestExecuteWithCustomSignal(t *testing.T) {
	var ctxErr error
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			p, _ := os.FindProcess(os.Getpid())
			p.Signal(syscall.SIGUSR1)

			select {
			case <-cmd.Context().Done():
				ctxErr = cmd.Context().Err()
			case <-time.After(time.Second):
				t.Fatal("context was not cancelled after SIGUSR1")
			}
			return nil
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	if err := ExecuteE(cmd, WithSignals(syscall.SIGUSR1)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctxErr == nil {
		t.Fatal("expected context to be cancelled")
	}
}

func TestContextCancelStopsListening(t *testing.T) {
	_, cancel := Context()
	cancel()
	// Should not panic or leak; verifies cleanup runs cleanly.
}
