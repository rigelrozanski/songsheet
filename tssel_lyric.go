package main

import (
	"errors"
	"fmt"
	"strings"
)

type lyrics struct {
	lyrics string
}

var _ tssElement = lyrics{}

func (s lyrics) parseText(lines []string) (reduced []string, elem tssElement, err error) {
	if len(lines) < 1 {
		return lines, elem,
			fmt.Errorf("improper number of input lines,"+
				" want 1 have %v", len(lines))
	}

	sll := lyrics{}
	sll.lyrics = lines[0]
	return lines[1:], sll, nil
}

func (s lyrics) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {

	// accumulate all the used height as it's used
	usedHeight := 0.0

	// print the lyric
	pdf.SetFont("courier", "", lyricFontPt)
	fontH := GetFontHeight(lyricFontPt)
	fontW := GetCourierFontWidthFromHeight(fontH)
	xLyricStart := bnd.left - fontW/2 // - because of slight right shift in sine annotations

	// 3/2 because of tall characters extending beyond height calculation
	yLyric := bnd.top + 1.3*fontH

	// the lyrics could just be printed in one go,
	// however do to the inaccuracies of determining
	// font heights and widths (boohoo) it will look
	// better to just print out each char individually
	for i, ch := range s.lyrics {
		xLyric := xLyricStart + float64(i)*fontW
		pdf.Text(xLyric, yLyric, string(ch))
	}
	usedHeight += 1.3 * fontH
	return bounds{bnd.top + usedHeight, bnd.left, bnd.bottom, bnd.right}
}

func determineLyricFontPt(
	lines []string, bnd bounds) (maxHumps, fontPt float64, err error) {

	// find the longest set of humps among them all
	humpsChars := 0
	for i := 0; i < len(lines)-1; i++ {
		if strings.HasPrefix(lines[i], "_") &&
			strings.HasPrefix(lines[i+1], " \\_/") {

			humpsCharsNew := len(strings.TrimSpace(lines[i]))

			// +1 for the leading space just trimmed
			secondLineLen := len(strings.TrimSpace(lines[i+1])) + 1

			if humpsCharsNew < secondLineLen {
				humpsCharsNew = secondLineLen
			}
			if humpsCharsNew > humpsChars {
				humpsChars = humpsCharsNew
			}
		}
	}
	if humpsChars == 0 {
		return 0, 0, errors.New("could not find a sine curve to determine the lyric font pt")
	}

	xStart := bnd.left
	xEnd := bnd.right - padding
	width := xEnd - xStart
	fontWidth := width / float64(humpsChars)
	fontHeight := GetCourierFontHeightFromWidth(fontWidth)
	return float64(humpsChars) / charsToaHump, GetFontPt(fontHeight), nil
}
