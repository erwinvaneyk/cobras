package cobras

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

type Options interface {
	Complete(cmd *cobra.Command, args []string) error
	Validate() error
	Run(ctx context.Context) error
}

type RunOption func(*config)

type config struct {
	signals []os.Signal
}

func defaultConfig() config {
	return config{
		signals: []os.Signal{os.Interrupt},
	}
}

func applyOptions(options []RunOption) config {
	cfg := defaultConfig()
	for _, o := range options {
		o(&cfg)
	}
	return cfg
}

// WithSignals overrides which signals cancel the context.
// Pass no arguments to disable signal handling.
func WithSignals(signals ...os.Signal) RunOption {
	return func(c *config) {
		c.signals = signals
	}
}

func Run(opts Options, options ...RunOption) func(cmd *cobra.Command, args []string) {
	cfg := applyOptions(options)
	return func(cmd *cobra.Command, args []string) {
		err := opts.Complete(cmd, args)
		if err != nil {
			printErrorAndDie(err)
		}

		err = opts.Validate()
		if err != nil {
			printErrorAndDie(err)
		}

		ctx := cmd.Context()
		if len(cfg.signals) > 0 {
			var cancel context.CancelFunc
			ctx, cancel = signal.NotifyContext(ctx, cfg.signals...)
			defer cancel()
		}

		err = opts.Run(ctx)
		if err != nil {
			printErrorAndDie(err)
		}
	}
}

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

func Execute(cmd *cobra.Command, options ...RunOption) {
	cfg := applyOptions(options)
	ctx, cancel := Context(cfg.signals...)
	defer cancel()
	err := cmd.ExecuteContext(ctx)
	if err != nil {
		printErrorAndDie(err)
	}
}

func printErrorAndDie(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
