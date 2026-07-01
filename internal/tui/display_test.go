package tui

import "testing"

func TestClampSelection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		cursor, count, want int
	}{
		{0, 0, 0},
		{0, 3, 0},
		{2, 3, 2},
		{5, 3, 2},
		{-1, 3, 0},
	}
	for _, tc := range cases {
		if got := ClampSelection(tc.cursor, tc.count); got != tc.want {
			t.Fatalf("ClampSelection(%d,%d)=%d want %d", tc.cursor, tc.count, got, tc.want)
		}
	}
}
