package main

import "strings"

const (
	commentPrefix = "//"
)

func deleteComments(lines []string) (out []string) {
LOOP:
	for _, line := range lines {
		switch {
		// do not include this line
		case strings.HasPrefix(line, commentPrefix):
			continue LOOP

		// take everything before the comment
		case strings.Contains(line, commentPrefix):
			splt := strings.SplitN(line, commentPrefix, 2)
			if len(splt) != 2 {
				panic("something wrong with strings library")
			}
			out = append(out, splt[0])
			continue LOOP
		default:
			out = append(out, line)
		}
	}
	return out
}
