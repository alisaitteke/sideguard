package tray

import (
	"testing"

	"github.com/alisaitteke/sideguard/internal/api"
)

func TestDetectNewPending(t *testing.T) {
	t.Parallel()

	item := func(id string) api.PendingApproval {
		return api.PendingApproval{ID: id}
	}

	tests := []struct {
		name string
		prev []api.PendingApproval
		next []api.PendingApproval
		want bool
	}{
		{
			name: "empty to empty",
			prev: nil,
			next: nil,
			want: false,
		},
		{
			name: "both empty slices",
			prev: []api.PendingApproval{},
			next: []api.PendingApproval{},
			want: false,
		},
		{
			name: "same IDs same count",
			prev: []api.PendingApproval{item("a"), item("b")},
			next: []api.PendingApproval{item("a"), item("b")},
			want: false,
		},
		{
			name: "reordered same IDs",
			prev: []api.PendingApproval{item("a"), item("b")},
			next: []api.PendingApproval{item("b"), item("a")},
			want: false,
		},
		{
			name: "new ID added",
			prev: []api.PendingApproval{item("a")},
			next: []api.PendingApproval{item("a"), item("b")},
			want: true,
		},
		{
			name: "empty to non-empty",
			prev: nil,
			next: []api.PendingApproval{item("a")},
			want: true,
		},
		{
			name: "count decreased after decide",
			prev: []api.PendingApproval{item("a"), item("b")},
			next: []api.PendingApproval{item("a")},
			want: false,
		},
		{
			name: "count same ID replaced",
			prev: []api.PendingApproval{item("a")},
			next: []api.PendingApproval{item("b")},
			want: true,
		},
		{
			name: "prev non-empty next nil",
			prev: []api.PendingApproval{item("a")},
			next: nil,
			want: false,
		},
		{
			name: "daemon down next nil with prev",
			prev: []api.PendingApproval{item("a"), item("b")},
			next: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DetectNewPending(tt.prev, tt.next); got != tt.want {
				t.Fatalf("DetectNewPending() = %v, want %v", got, tt.want)
			}
		})
	}
}
