package main

import (
	"errors"
	"fmt"
	"strings"
)

type spacer struct{}

var _ tssElement = spacer{}

func (s spacer) parseText(lines []string) (reduced []string, elem tssElement, err error) {
	if len(lines) < 1 {
		return lines, elem,
			fmt.Errorf("improper number of input lines, want 1 have %v", len(lines))
	}
	if len(strings.TrimSpace(lines[0])) != 0 {
		return lines, elem, errors.New("blank line contains content")
	}
	return lines[1:], spacer{}, nil
}

func (s spacer) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {
	lineHeight := GetFontHeight(lyricFontPt) * spacingRatioFlag
	return bounds{bnd.top + lineHeight, bnd.left, bnd.bottom, bnd.right}
}
