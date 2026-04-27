package display

import "testing"

func TestSparkBar(t *testing.T) {
	tests := []struct {
		name  string
		value int
		max   int
		want  string
	}{
		{name: "zero max", value: 100, max: 0, want: ""},
		{name: "full bar", value: 100, max: 100, want: "████████████████████"},
		{name: "half bar", value: 50, max: 100, want: "██████████░░░░░░░░░░"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SparkBar(tt.value, tt.max); got != tt.want {
				t.Fatalf("SparkBar() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrendArrow(t *testing.T) {
	tests := []struct {
		name string
		prev int
		curr int
		want string
	}{
		{name: "zero previous", prev: 0, curr: 100, want: " "},
		{name: "increase", prev: 100, curr: 110, want: "↑"},
		{name: "decrease", prev: 100, curr: 90, want: "↓"},
		{name: "stable", prev: 100, curr: 101, want: "→"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrendArrow(tt.prev, tt.curr); got != tt.want {
				t.Fatalf("TrendArrow() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPaceArrow(t *testing.T) {
	tests := []struct {
		name string
		prev float64
		curr float64
		want string
	}{
		{name: "lower pace is improvement", prev: 180, curr: 170, want: "↑"},
		{name: "higher pace is regression", prev: 170, curr: 180, want: "↓"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PaceArrow(tt.prev, tt.curr); got != tt.want {
				t.Fatalf("PaceArrow() = %q, want %q", got, tt.want)
			}
		})
	}
}
