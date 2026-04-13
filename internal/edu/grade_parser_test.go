package edu

import "testing"

func TestParseGradeFromLogLines(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  int
		found bool
	}{
		{
			name:  "single grade command",
			lines: []string{"some output", "::edu-grade::85", "more output"},
			want:  85,
			found: true,
		},
		{
			name:  "multiple grade commands takes last",
			lines: []string{"::edu-grade::50", "::edu-grade::92"},
			want:  92,
			found: true,
		},
		{
			name:  "no grade command",
			lines: []string{"all tests passed", "done"},
			want:  0,
			found: false,
		},
		{
			name:  "grade zero",
			lines: []string{"::edu-grade::0"},
			want:  0,
			found: true,
		},
		{
			name:  "grade 100",
			lines: []string{"::edu-grade::100"},
			want:  100,
			found: true,
		},
		{
			name:  "grade over 100 ignored",
			lines: []string{"::edu-grade::101"},
			want:  0,
			found: false,
		},
		{
			name:  "grade negative ignored",
			lines: []string{"::edu-grade::-5"},
			want:  0,
			found: false,
		},
		{
			name:  "grade non-numeric ignored",
			lines: []string{"::edu-grade::abc"},
			want:  0,
			found: false,
		},
		{
			name:  "grade in middle of line",
			lines: []string{"prefix ::edu-grade::75 suffix"},
			want:  75,
			found: true,
		},
		{
			name:  "empty lines",
			lines: []string{},
			want:  0,
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := ParseGradeFromLogLines(tt.lines)
			if found != tt.found {
				t.Errorf("ParseGradeFromLogLines() found = %v, want %v", found, tt.found)
			}
			if got != tt.want {
				t.Errorf("ParseGradeFromLogLines() got = %v, want %v", got, tt.want)
			}
		})
	}
}
