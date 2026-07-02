package tui

import (
	"strings"
	"testing"

	"github.com/alisaitteke/sideguard/internal/api"
)

func TestRunRequiresTTY(t *testing.T) {
	t.Parallel()
	err := Run(api.NewClient(), Options{})
	if err == nil {
		t.Fatal("expected error when stdin is not a TTY")
	}
	if !strings.Contains(err.Error(), "TTY") {
		t.Fatalf("unexpected error: %v", err)
	}
}
