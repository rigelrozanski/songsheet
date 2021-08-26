package main

import (
	"fmt"
	"strings"
	"unicode"
)

// TODO
// - chord chart to be able to sqeeze a few more chords in beyond
//   the standard spacing
// - chord chart to be able to go across the top if the squeeze doesn't work
// - eliminate cactus prickles where no label exists

type chordChart struct {
	chords          []Chord
	labelFontPt     float64
	positionsFontPt float64
}

type Chord struct {
	name      string   // must be 1 or 2 characters
	positions []string // from thick to thin guitar strings
}

var _ tssElement = chordChart{}

func (c chordChart) parseText(lines []string) (reduced []string, elem tssElement, err error) {
	if len(lines) < 9 {
		return lines, elem,
			fmt.Errorf("improper number of input lines,"+
				" want at least 9 have %v", len(lines))
	}

	// checking form, must be in the pattern as such:
	//  |  |  |
	//- 1  3
	//- 0  2
	//- 3  0
	//- 0  0
	//- 1  1
	//- 0  0
	//  |  |  |
	//  F  G  C
	if !strings.HasPrefix(lines[0], "  |  |  |") {
		return lines, elem, fmt.Errorf("not a chord chart (line 1)")
	}
	if !strings.HasPrefix(lines[7], "  |  |  |") {
		return lines, elem, fmt.Errorf("not a chord chart (line 7)")
	}
	for i := 1; i <= 6; i++ {
		if !strings.HasPrefix(lines[i], "- ") {
			return lines, elem, fmt.Errorf("not a chord chart (line %v)", i)
		}
	}

	cOut := chordChart{
		labelFontPt:     12,
		positionsFontPt: 10,
	}
	// get the chords
	chordNames := lines[8]
	for j := 2; j < len(chordNames); j += 3 {

		if chordNames[j] == ' ' {
			// this chord is not labelled, must be the end of the chords
			break
		}

		newChord := Chord{name: string(chordNames[j])}

		// add the second and third character to the name (if it exists)
		if j+1 < len(chordNames) && chordNames[j+1] != ' ' {
			newChord.name += string(chordNames[j+1])
			if j+2 < len(chordNames) && chordNames[j+2] != ' ' {
				newChord.name += string(chordNames[j+2])
			}
		}

		// add all the guitar strings
		for i := 1; i <= 6; i++ {
			word := string(lines[i][j])

			if j+1 < len(lines[i]) {
				if lines[i][j+1] != ' ' {
					word += string(lines[i][j+1])
				}
			}
			newChord.positions = append(newChord.positions, word)
		}
		cOut.chords = append(cOut.chords, newChord)
	}

	// chop off the first 9 lines
	return lines[9:], cOut, nil
}

// test to see whether or not the second and third inputs are
// superscript and/subscript to the first input if it is a chord
func determineChordsSubscriptSuperscriptSlide(ch1, ch2, ch3, ch4 rune) (subscript, superscript rune, slide bool) {
	if !(unicode.IsLetter(ch1) && unicode.IsUpper(ch1)) {
		return ' ', ' ', false
	}
	slide = false
	subscript, superscript = ' ', ' '
	if unicode.IsNumber(ch2) || (unicode.IsLetter(ch2) && unicode.IsLower(ch2)) {
		subscript = ch2
	}
	if unicode.IsNumber(ch3) || (unicode.IsLetter(ch3) && unicode.IsLower(ch3)) {
		superscript = ch3
	}

	if (subscript != ' ' && (ch3 == '/' || ch3 == '\\')) ||
		(superscript != ' ' && (ch4 == '/' || ch4 == '\\')) {

		slide = true
	}

	return subscript, superscript, slide
}

func (c chordChart) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {

	usedHeight := 0.0

	// the top zone of the pillar that shows the guitar string thicknesses
	thicknessIndicatorMargin := padding / 2

	spacing := padding / 2
	cactusZoneWidth := 0.0
	cactusPrickleSpacing := padding
	cactusZoneWidth = padding // one for the cactus

	noLines := len(thicknesses)

	// print thicknesses and decorations around them
	var xStart, xEnd, y float64

	// decoration params
	melodyFontPt := lyricFontPt
	melodyFontH := GetFontHeight(melodyFontPt)
	melodyFontW := GetCourierFontWidthFromHeight(melodyFontH)
	melodyHPadding := melodyFontH * 0.3

	for i := 0; i < noLines; i++ {
		// thicknesses
		pdf.SetLineWidth(thicknesses[i])
		y = bnd.top + cactusZoneWidth + (float64(i) * spacing)
		xStart = bnd.left
		xEnd = xStart + thicknessIndicatorMargin
		pdf.Line(xStart, y, xEnd, y)

		// decorations
		switch i {
		case 2, 3:
			xMod := xStart + thicknessIndicatorMargin/2
			yMod := y + spacing/3
			if i == 2 { // above
				yMod = y - spacing/3
			}
			pdf.Circle(xMod, yMod, melodyHPadding/1.5, "F")
		case 1, 4:
			xMar := (xEnd - xStart - melodyFontW) / 2
			xModStart := xStart + xMar
			xModEnd := xStart + xMar + melodyFontW
			yMod := y + spacing/3
			if i == 1 { // above
				yMod = y - spacing/3
			}
			pdf.SetLineWidth(thinishLW)
			pdf.Line(xModStart, yMod, xModEnd, yMod)
		case 0, 5:
			xMar := (xEnd - xStart - melodyFontW) / 2
			xModStart := xStart + xMar
			xModEnd := xStart + xMar + melodyFontW
			xModMid := (xModStart + xModEnd) / 2
			yMod := y + spacing/4 + melodyHPadding
			yModMid := yMod - melodyHPadding*2
			if i == 0 { // above
				yMod = y - spacing/4 - melodyHPadding
				yModMid = yMod + melodyHPadding*2
			}
			pdf.SetLineWidth(thinishLW)
			pdf.Curve(xModStart, yMod, xModMid, yModMid, xModEnd, yMod, "")
		}
	}
	usedHeight += cactusZoneWidth + float64(noLines)*spacing

	// print seperator
	pdf.SetLineWidth(thinestLW)
	yStart := bnd.top + cactusZoneWidth
	yEnd := yStart + float64(noLines-1)*spacing
	xStart = bnd.left + thicknessIndicatorMargin
	xEnd = xStart
	pdf.Line(xStart, yStart, xEnd, yEnd)

	// print pillar lines
	for i := 0; i < noLines; i++ {
		pdf.SetLineWidth(thinestLW)
		y = bnd.top + cactusZoneWidth + (float64(i) * spacing)
		xStart = bnd.left + thicknessIndicatorMargin
		xEnd = bnd.right - padding
		pdf.Line(xStart, y, xEnd, y)
	}

	// print prickles
	xStart = bnd.left + thicknessIndicatorMargin + cactusPrickleSpacing/2
	xEnd = bnd.right - padding
	chordIndex := 0
	pdf.SetFont("courier", "", c.labelFontPt)
	fontHeight := GetFontHeight(c.labelFontPt)
	labelPadding := fontHeight * 0.1
	fontWidth := GetCourierFontWidthFromHeight(fontHeight)
	for x := xStart; x < xEnd; x += cactusPrickleSpacing {
		pdf.SetLineWidth(thinestLW)
		yTopStart := bnd.top
		yTopEnd := yTopStart + cactusZoneWidth/2
		yBottomStart := bnd.top + cactusZoneWidth +
			(float64(noLines-1) * spacing) + cactusZoneWidth/2
		yBottomEnd := yBottomStart + cactusZoneWidth/2

		pdf.Line(x, yTopStart, x, yTopEnd)
		pdf.Line(x, yBottomStart, x, yBottomEnd)

		// print labels
		if chordIndex >= len(c.chords) {
			continue
		}
		chd := c.chords[chordIndex]

		ch1, ch2, ch3 := ' ', ' ', ' '
		switch len(chd.name) {
		case 3:
			ch3 = rune(chd.name[2])
			fallthrough
		case 2:
			ch2 = rune(chd.name[1])
			fallthrough
		case 1:
			ch1 = rune(chd.name[0])
		}

		subscriptCh, superscriptCh, _ := determineChordsSubscriptSuperscriptSlide(
			ch1, ch2, ch3, ' ')

		xLabel := x - fontWidth/2
		yLabel := yBottomEnd + fontHeight + labelPadding
		pdf.SetFont("courier", "", c.labelFontPt)
		pdf.Text(xLabel, yLabel, string(ch1))

		if subscriptCh != ' ' {
			pdf.SetFont("courier", "", c.labelFontPt*subsupSizeMul)
			pdf.Text(xLabel+fontWidth, yLabel, string(subscriptCh))
		}
		if superscriptCh != ' ' {
			panic("chords labels cannot have superscript")
		}

		// print positions
		pdf.SetFont("courier", "", c.positionsFontPt)
		posFontH := GetFontHeight(c.positionsFontPt)
		posFontW := GetCourierFontWidthFromHeight(posFontH)
		//xPositions := x - fontWidth/2 // maybe incorrect, but looks better
		xPositions := x - posFontW/2
		for i := 0; i < noLines; i++ {
			yPositions := bnd.top + cactusZoneWidth +
				(float64(i) * spacing) + posFontH/2

			if chd.positions[i] == "x" {
				ext := posFontW / 2
				y := yPositions - posFontH/2
				pdf.Line(x-ext, y-ext, x+ext, y+ext)
				pdf.Line(x-ext, y+ext, x+ext, y-ext)
				continue
			}

			pdf.Text(xPositions, yPositions, chd.positions[i])
		}

		chordIndex++
	}
	// for the lower prickles and labels
	// (upper prickles already accounted for in previous usedHeight accumulation)
	usedHeight += cactusZoneWidth + fontHeight + labelPadding

	return bounds{bnd.top + usedHeight, bnd.left, bnd.bottom, bnd.right}
}
