package edu

import (
	"regexp"
	"strconv"
)

var eduGradeRegex = regexp.MustCompile(`::edu-grade::(\d{1,3})`)

// ParseGradeFromLogLines scans log lines for the ::edu-grade::XX command.
// Returns the last valid grade (0-100) found and whether any was found.
func ParseGradeFromLogLines(lines []string) (int, bool) {
	lastGrade := 0
	found := false

	for _, line := range lines {
		matches := eduGradeRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			score, err := strconv.Atoi(m[1])
			if err != nil || score > 100 {
				continue
			}
			lastGrade = score
			found = true
		}
	}

	return lastGrade, found
}
