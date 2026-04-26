package collector

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCustomCollector_Collect(t *testing.T) {
	c := NewCustomCollector("echo hello")

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(data.Value) != "hello" {
		t.Errorf("expected 'hello', got %q", data.Value)
	}
}

func TestCustomCollector_Collect_CommandError(t *testing.T) {
	c := NewCustomCollector("exit 1")

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestCustomCollector_Collect_NonExistentCommand(t *testing.T) {
	c := NewCustomCollector("nonexistent_command_that_does_not_exist")

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected error for non-existent command")
	}
}

func TestCustomCollector_Collect_WithTimeout(t *testing.T) {
	// Command that takes longer than timeout
	c := &CustomCollector{
		command: "sleep 5",
		timeout: 100 * time.Millisecond,
	}

	ctx := context.Background()
	_, err := c.Collect(ctx)

	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestCustomCollector_Collect_OutputTrimming(t *testing.T) {
	c := NewCustomCollector("echo '  trimmed  '")

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Value != "trimmed" {
		t.Errorf("expected 'trimmed', got %q", data.Value)
	}
}

func TestCustomCollector_Collect_MultiLineOutput(t *testing.T) {
	c := NewCustomCollector("printf 'line1\nline2\nline3'")

	ctx := context.Background()
	data, err := c.Collect(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Multi-line output should be preserved
	if !strings.Contains(data.Value, "line1") || !strings.Contains(data.Value, "line3") {
		t.Errorf("expected multi-line output, got %q", data.Value)
	}
}
