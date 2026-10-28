package gliffy

import (
	"fmt"
	"testing"
)

func TestArrowStyle(t *testing.T) {
	tests := []struct {
		id   *int
		name string
		fill int
		size int
	}{
		{nil, "none", 0, 6},
		{new(-1), "none", 0, 6},
		{new(20), "none", 0, 6},
		{new(0), "none", 0, 6},
		{new(1), "open", 0, 6},
		{new(2), "block", 1, 6},
		{new(3), "block", 0, 6},
		{new(4), "block", 0, 10},
		{new(5), "diamond", 0, 12},
		{new(6), "open", 0, 10},
		{new(7), "diamond", 1, 12},
		{new(8), "classic", 1, 6},
		{new(9), "ERzeroToMany", 1, 10},
		{new(10), "ERoneToMany", 1, 10},
		{new(11), "ERmandOne", 1, 10},
		{new(12), "ERone", 1, 10},
		{new(13), "ERzeroToOne", 1, 10},
		{new(14), "ERmany", 1, 10},
		{new(15), "oval", 0, 10},
		{new(16), "dash", 0, 6},
		{new(17), "block", 1, 6},
		{new(18), "classic", 1, 6},
		{new(19), "openAsync", 0, 10},
	}
	for _, tt := range tests {
		for _, start := range []bool{false, true} {
			prefix, terminal := "end", "target"
			if start {
				prefix, terminal = "start", "source"
			}
			want := fmt.Sprintf("%sArrow=%s;%sFill=%d;%sSize=%d;", prefix, tt.name, prefix, tt.fill, prefix, tt.size)
			if tt.id != nil && *tt.id == 15 {
				want += terminal + "PerimeterSpacing=6;"
			}
			if got := arrowStyle(tt.id, start); got != want {
				t.Errorf("arrowStyle(%v, %t) = %q, want %q", tt.id, start, got, want)
			}
		}
	}
}
