// Package cobras provides helpers for building Cobra-based CLIs with
// signal-aware context management and a structured command lifecycle.
package cobras

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

// Options defines a three-phase command lifecycle:
//
//  1. Complete binds command-line flags and positional arguments to the receiver.
//  2. Validate checks that the bound values form a valid configuration.
//  3. Run executes the command logic with a context that is canceled on
//     the configured OS signals (default: SIGINT).
//
// Use [Run] or [RunE] to wire an Options implementation into a [cobra.Command].
type Options interface {
	Complete(cmd *cobra.Command, args []string) error
	Validate() error
	Run(ctx context.Context) error
}

// ExecuteOption configures the behavior of [Execute].
type ExecuteOption func(*config)

type config struct {
	signals []os.Signal
}

func defaultConfig() config {
	return config{
		// Defaults to SIGINT for backward compatibility
		signals: []os.Signal{os.Interrupt},
	}
}

func applyOptions(options []ExecuteOption) config {
	cfg := defaultConfig()
	for _, o := range options {
		o(&cfg)
	}
	return cfg
}

// WithSignals overrides which signals cancel the context.
// Pass no arguments to disable signal handling.
func WithSignals(signals ...os.Signal) ExecuteOption {
	return func(c *config) {
		c.signals = signals
	}
}

// Run returns a function compatible with [cobra.Command.Run] that drives the
// Complete→Validate→Run lifecycle on opts. If any phase returns an error the
// process prints it to stderr and exits with code 1.
//
// The command's context (set by [Execute] or [cobra.Command.ExecuteContext]) is
// passed through to opts.Run.
func Run(opts Options) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		err := opts.Complete(cmd, args)
		if err != nil {
			printErrorAndDie(err)
		}

		err = opts.Validate()
		if err != nil {
			printErrorAndDie(err)
		}

		err = opts.Run(cmd.Context())
		if err != nil {
			printErrorAndDie(err)
		}
	}
}

// RunE returns a function compatible with [cobra.Command.RunE] that drives the
// Complete→Validate→Run lifecycle on opts. Unlike [Run], errors are returned
// to the caller instead of printing and exiting.
func RunE(opts Options) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := opts.Complete(cmd, args); err != nil {
			return err
		}
		if err := opts.Validate(); err != nil {
			return err
		}
		return opts.Run(cmd.Context())
	}
}

// Context creates a background context that is cancelled when one of the given
// signals is received.
//
// If no signals are provided it defaults to SIGINT for backward compatibility.
// The returned cancel function stops signal listening and cancels the context.
func Context(signals ...os.Signal) (ctx context.Context, cancel func()) {
	if len(signals) == 0 {
		signals = defaultConfig().signals
	}
	ctx, origCancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	go func() {
		select {
		case <-c:
			origCancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() {
		signal.Stop(c)
		origCancel()
	}
}

// ExecuteE is a convenience entrypoint that creates a signal-aware context and
// calls [cobra.Command.ExecuteContext]. Unlike [Execute], errors are returned
// to the caller instead of printing and exiting.
func ExecuteE(cmd *cobra.Command, options ...ExecuteOption) error {
	cfg := applyOptions(options)
	ctx, cancel := Context(cfg.signals...)
	defer cancel()
	return cmd.ExecuteContext(ctx)
}

// Execute is a convenience entrypoint that creates a signal-aware context and
// calls [cobra.Command.ExecuteContext]. If execution fails the process prints
// the error to stderr and exits with code 1.
func Execute(cmd *cobra.Command, options ...ExecuteOption) {
	if err := ExecuteE(cmd, options...); err != nil {
		printErrorAndDie(err)
	}
}

func printErrorAndDie(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
