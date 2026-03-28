package cobras

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if len(cfg.signals) != 1 {
		t.Fatalf("expected 1 default signal, got %d", len(cfg.signals))
	}
	if cfg.signals[0] != os.Interrupt {
		t.Fatalf("expected os.Interrupt, got %v", cfg.signals[0])
	}
}

func TestWithSignals(t *testing.T) {
	cfg := applyOptions([]RunOption{WithSignals(syscall.SIGTERM, syscall.SIGHUP)})
	if len(cfg.signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(cfg.signals))
	}
	if cfg.signals[0] != syscall.SIGTERM || cfg.signals[1] != syscall.SIGHUP {
		t.Fatalf("unexpected signals: %v", cfg.signals)
	}
}

func TestWithSignalsEmpty(t *testing.T) {
	cfg := applyOptions([]RunOption{WithSignals()})
	if len(cfg.signals) != 0 {
		t.Fatalf("expected 0 signals, got %d", len(cfg.signals))
	}
}

func TestContextDefaultCancelsOnInterrupt(t *testing.T) {
	ctx, cancel := Context()
	defer cancel()

	// Send ourselves SIGINT
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(os.Interrupt)

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled after SIGINT")
	}
}

func TestContextWithCustomSignal(t *testing.T) {
	ctx, cancel := Context(syscall.SIGUSR1)
	defer cancel()

	p, _ := os.FindProcess(os.Getpid())
	p.Signal(syscall.SIGUSR1)

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled after SIGUSR1")
	}
}

func TestRunWithNoSignals(t *testing.T) {
	// WithSignals() on Run disables signal handling
	cfg := applyOptions([]RunOption{WithSignals()})
	if len(cfg.signals) != 0 {
		t.Fatalf("expected 0 signals, got %d", len(cfg.signals))
	}
}

func TestContextCancelStopsListening(t *testing.T) {
	_, cancel := Context()
	cancel()
	// Should not panic or leak; verifies cleanup runs cleanly.
}
