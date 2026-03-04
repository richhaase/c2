// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package display

import "testing"

func TestFormatMeters(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{1000, "1,000"},
		{1_000_000, "1,000,000"},
		{12345, "12,345"},
	}
	for _, tt := range tests {
		got := FormatMeters(tt.input)
		if got != tt.want {
			t.Errorf("FormatMeters(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0.0, "0.0%"},
		{0.5, "50.0%"},
		{1.0, "100.0%"},
		{0.1234, "12.3%"},
	}
	for _, tt := range tests {
		got := FormatPercent(tt.input)
		if got != tt.want {
			t.Errorf("FormatPercent(%f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
