package main

import (
	"fmt"
	"unicode"
)

// TODO
//  - extras for melody
//  - '_' for steadyness (streches beyond note)
//  - 'v' for vibrato
//  - 'V' for intense vibrato
//  - '|' for halting singing
//  - ability to combine '_', 'v', 'V' and '|'

type melodies []melody

var _ tssElement = melodies{}

type melody struct {
	num                rune
	modifierIsAboveNum bool // otherwise below
	modifier           rune // either '.', '-', or '~'

	// whether to display:
	//  '(' = the number with tight brackets
	//  '/' = to add a modfier for a slide up
	//  '\' = to add a modifier for a slide down
	extra rune
}

// NewMelody creates a new melody object
func NewMelody(num rune, modifierIsAboveNum bool, modifier, extra rune) melody {
	return melody{
		num:                num,
		modifierIsAboveNum: modifierIsAboveNum,
		modifier:           modifier,
		extra:              extra,
	}
}

func NewMelodyFromTwoChars(ch1, ch2 rune) (m melody, success bool) {

	modifierIsAboveNum := false
	num, modifier := ' ', ' '
	switch {
	case runeIsMod(ch1) && unicode.IsNumber(ch2):
		num = ch2
		modifier = ch1
		modifierIsAboveNum = false
	case runeIsMod(ch2) && unicode.IsNumber(ch1):
		num = ch1
		modifier = ch2
		modifierIsAboveNum = true
	default:
		return m, false
	}

	return melody{
		num:                num,
		modifierIsAboveNum: modifierIsAboveNum,
		modifier:           modifier,
		extra:              ' ',
	}, true
}

func (m melody) print(pdf Pdf, x, y float64, fontH float64, nextMelodyNum rune, xNumNext float64) {
	if m.num == ' ' {
		return
	}

	fontW := GetCourierFontWidthFromHeight(fontH)

	// print the melodies
	melodyFontPt := lyricFontPt
	pdf.SetFont("courier", "", melodyFontPt)
	melodyFontH := GetFontHeight(melodyFontPt)
	melodyFontW := GetCourierFontWidthFromHeight(melodyFontH)
	melodyHPadding := melodyFontH * 0.3

	// print number
	pdf.Text(x, y, string(m.num))

	// print modifier
	switch m.modifier {
	case mod1:
		xMod := x + melodyFontW/2
		yMod := y - melodyHPadding*0.25
		if m.modifierIsAboveNum {
			yMod = y - melodyFontH + 0.65*melodyHPadding
		}
		pdf.Circle(xMod, yMod, melodyHPadding/2.0, "F")
	case mod2:
		xModStart := x
		xModEnd := x + melodyFontW
		yMod := y - 0.6*melodyHPadding
		if m.modifierIsAboveNum {
			yMod = y - melodyFontH + 1.2*melodyHPadding
		}
		pdf.SetLineWidth(thinishLW)
		pdf.Line(xModStart, yMod, xModEnd, yMod)
	case mod3:
		xModStart := x
		xModMid := x + melodyFontW/2
		xModEnd := x + melodyFontW
		yMod := y + 0.1*melodyHPadding
		yModMid := yMod - melodyHPadding*2
		if m.modifierIsAboveNum {
			yMod = y - melodyFontH + 0.36*melodyHPadding
			yModMid = yMod + melodyHPadding*2
		}
		pdf.SetLineWidth(thinishLW)
		pdf.Curve(xModStart, yMod, xModMid, yModMid, xModEnd, yMod, "")
	default:
		panic(fmt.Errorf("unknown modifier %v", m.modifier))
	}

	// print extra decorations

	switch m.extra {
	case extraBrac:
		xBrac1 := x - fontW*0.5
		xBrac2 := x + fontW*0.5
		yBrac := y - melodyHPadding/2 // shift for looks
		pdf.Text(xBrac1, yBrac, "(")
		pdf.Text(xBrac2, yBrac, ")")

	case extraSldUp:
		draw := true

		// offset factors per number (specific to font)
		XStart0, YStart0, XEnd0, YEnd0 := 0.55, 0.10, 0.45, 0.60
		XStart1, YStart1, XEnd1, YEnd1 := 0.75, 0.14, 0.30, 0.75
		XStart2, YStart2, XEnd2, YEnd2 := 0.75, 0.14, 0.30, 0.80
		XStart3, YStart3, XEnd3, YEnd3 := 0.65, 0.14, 0.30, 0.75
		XStart4, YStart4, XEnd4, YEnd4 := 0.75, 0.14, 0.60, 0.53
		XStart5, YStart5, XEnd5, YEnd5 := 0.65, 0.14, 0.30, 0.60
		XStart6, YStart6, XEnd6, YEnd6 := 0.65, 0.14, 0.50, 0.65
		XStart7, YStart7, XEnd7, YEnd7 := 0.50, 0.10, 0.30, 0.65
		XStart8, YStart8, XEnd8, YEnd8 := 0.65, 0.14, 0.30, 0.65
		XStart9, YStart9, XEnd9, YEnd9 := 0.55, 0.25, 0.30, 0.65

		// determine the starting location
		xSldStart, ySldStart := 0.0, 0.0
		switch m.num {
		case '0':
			xSldStart = x + fontW*XStart0
			ySldStart = y - melodyHPadding*YStart0
		case '1':
			xSldStart = x + fontW*XStart1
			ySldStart = y - melodyHPadding*YStart1
		case '2':
			xSldStart = x + fontW*XStart2
			ySldStart = y - melodyHPadding*YStart2
		case '3':
			xSldStart = x + fontW*XStart3
			ySldStart = y - melodyHPadding*YStart3
		case '4':
			xSldStart = x + fontW*XStart4
			ySldStart = y - melodyHPadding*YStart4
		case '5':
			xSldStart = x + fontW*XStart5
			ySldStart = y - melodyHPadding*YStart5
		case '6':
			xSldStart = x + fontW*XStart6
			ySldStart = y - melodyHPadding*YStart6
		case '7':
			xSldStart = x + fontW*XStart7
			ySldStart = y - melodyHPadding*YStart7
		case '8':
			xSldStart = x + fontW*XStart8
			ySldStart = y - melodyHPadding*YStart8
		case '9':
			xSldStart = x + fontW*XStart9
			ySldStart = y - melodyHPadding*YStart9
		default:
			draw = false
		}

		// determine the ending location
		xSldEnd, ySldEnd := 0.0, 0.0
		switch nextMelodyNum {
		case '0':
			xSldEnd = xNumNext + fontW*XEnd0
			ySldEnd = y - fontH + melodyHPadding*YEnd0
		case '1':
			xSldEnd = xNumNext + fontW*XEnd1
			ySldEnd = y - fontH + melodyHPadding*YEnd1
		case '2':
			xSldEnd = xNumNext + fontW*XEnd2
			ySldEnd = y - fontH + melodyHPadding*YEnd2
		case '3':
			xSldEnd = xNumNext + fontW*XEnd3
			ySldEnd = y - fontH + melodyHPadding*YEnd3
		case '4':
			xSldEnd = xNumNext + fontW*XEnd4
			ySldEnd = y - fontH + melodyHPadding*YEnd4
		case '5':
			xSldEnd = xNumNext + fontW*XEnd5
			ySldEnd = y - fontH + melodyHPadding*YEnd5
		case '6':
			xSldEnd = xNumNext + fontW*XEnd6
			ySldEnd = y - fontH + melodyHPadding*YEnd6
		case '7':
			xSldEnd = xNumNext + fontW*XEnd7
			ySldEnd = y - fontH + melodyHPadding*YEnd7
		case '8':
			xSldEnd = xNumNext + fontW*XEnd8
			ySldEnd = y - fontH + melodyHPadding*YEnd8
		case '9':
			xSldEnd = xNumNext + fontW*XEnd9
			ySldEnd = y - fontH + melodyHPadding*YEnd9
		default:
			draw = false
		}

		if draw {
			pdf.SetLineCapStyle("round")
			defer pdf.SetLineCapStyle("")
			pdf.SetLineWidth(thinishtLW)
			pdf.Line(xSldStart, ySldStart, xSldEnd, ySldEnd)
		}

	case extraSldDown:
		draw := true

		// offset factors per number (specific to font)
		XStart0, YStart0, XEnd0, YEnd0 := 0.55, 0.60, 0.45, 0.10
		XStart1, YStart1, XEnd1, YEnd1 := 0.55, 0.65, 0.23, 0.25
		XStart2, YStart2, XEnd2, YEnd2 := 0.65, 0.64, 0.20, 0.25
		XStart3, YStart3, XEnd3, YEnd3 := 0.65, 0.64, 0.30, 0.20
		XStart4, YStart4, XEnd4, YEnd4 := 0.65, 0.55, 0.45, 0.20
		XStart5, YStart5, XEnd5, YEnd5 := 0.70, 0.67, 0.30, 0.10
		XStart6, YStart6, XEnd6, YEnd6 := 0.80, 0.64, 0.40, 0.15
		XStart7, YStart7, XEnd7, YEnd7 := 0.75, 0.60, 0.45, 0.10
		XStart8, YStart8, XEnd8, YEnd8 := 0.65, 0.64, 0.40, 0.10
		XStart9, YStart9, XEnd9, YEnd9 := 0.50, 0.53, 0.20, 0.15

		// determine the starting location
		xSldStart, ySldStart := 0.0, 0.0
		switch m.num {
		case '0':
			xSldStart = x + fontW*XStart0
			ySldStart = y - fontH + melodyHPadding*YStart0
		case '1':
			xSldStart = x + fontW*XStart1
			ySldStart = y - fontH + melodyHPadding*YStart1
		case '2':
			xSldStart = x + fontW*XStart2
			ySldStart = y - fontH + melodyHPadding*YStart2
		case '3':
			xSldStart = x + fontW*XStart3
			ySldStart = y - fontH + melodyHPadding*YStart3
		case '4':
			xSldStart = x + fontW*XStart4
			ySldStart = y - fontH + melodyHPadding*YStart4
		case '5':
			xSldStart = x + fontW*XStart5
			ySldStart = y - fontH + melodyHPadding*YStart5
		case '6':
			xSldStart = x + fontW*XStart6
			ySldStart = y - fontH + melodyHPadding*YStart6
		case '7':
			xSldStart = x + fontW*XStart7
			ySldStart = y - fontH + melodyHPadding*YStart7
		case '8':
			xSldStart = x + fontW*XStart8
			ySldStart = y - fontH + melodyHPadding*YStart8
		case '9':
			xSldStart = x + fontW*XStart9
			ySldStart = y - fontH + melodyHPadding*YStart9
		default:
			draw = false
		}

		// determine the ending location
		xSldEnd, ySldEnd := 0.0, 0.0
		switch nextMelodyNum {
		case '0':
			xSldEnd = xNumNext + fontW*XEnd0
			ySldEnd = y - melodyHPadding*YEnd0
		case '1':
			xSldEnd = xNumNext + fontW*XEnd1
			ySldEnd = y - melodyHPadding*YEnd1
		case '2':
			xSldEnd = xNumNext + fontW*XEnd2
			ySldEnd = y - melodyHPadding*YEnd2
		case '3':
			xSldEnd = xNumNext + fontW*XEnd3
			ySldEnd = y - melodyHPadding*YEnd3
		case '4':
			xSldEnd = xNumNext + fontW*XEnd4
			ySldEnd = y - melodyHPadding*YEnd4
		case '5':
			xSldEnd = xNumNext + fontW*XEnd5
			ySldEnd = y - melodyHPadding*YEnd5
		case '6':
			xSldEnd = xNumNext + fontW*XEnd6
			ySldEnd = y - melodyHPadding*YEnd6
		case '7':
			xSldEnd = xNumNext + fontW*XEnd7
			ySldEnd = y - melodyHPadding*YEnd7
		case '8':
			xSldEnd = xNumNext + fontW*XEnd8
			ySldEnd = y - melodyHPadding*YEnd8
		case '9':
			xSldEnd = xNumNext + fontW*XEnd9
			ySldEnd = y - melodyHPadding*YEnd9
		default:
			draw = false
		}

		if draw {
			pdf.SetLineCapStyle("round")
			defer pdf.SetLineCapStyle("")
			pdf.SetLineWidth(thinishtLW)
			pdf.Line(xSldStart, ySldStart, xSldEnd, ySldEnd)
		}

	}

}

// contains at least one number,
// and only numbers or spaces
func stringOnlyContainsNumbersAndSpaces(s string) bool {
	numFound := false
	for _, b := range s {
		r := rune(b)
		if !(unicode.IsSpace(r) || unicode.IsNumber(r)) {
			return false
		}
		if unicode.IsNumber(r) {
			numFound = true
		}
	}
	return numFound
}

const (
	mod1         = '.'
	mod2         = '-'
	mod3         = '~'
	extraBrac    = '('
	extraSldUp   = '\\'
	extraSldDown = '/'
)

func runeIsMod(r rune) bool {
	return r == mod1 || r == mod2 || r == mod3
}

func runeIsExtra(r rune) bool {
	return r == extraBrac || r == extraSldUp || r == extraSldDown
}

// contains at least one modifier,
// and only modifiers, extras, or spaces
func stringOnlyContainsMelodyModifiersAndExtras(s string) bool {
	found := false
	for _, b := range s {
		r := rune(b)
		if !(runeIsExtra(r) || runeIsMod(r) || unicode.IsSpace(r)) {
			return false
		}
		if runeIsExtra(r) || runeIsMod(r) {
			found = true
		}
	}
	return found
}

func (ms melodies) parseText(lines []string) (reduced []string, elem tssElement, err error) {
	if len(lines) < 2 {
		return lines, elem,
			fmt.Errorf("improper number of input lines,"+
				" want 1 have %v", len(lines))
	}

	// determine which lines should be used for the melody modifiers
	melodyNums, upper, lower := "", "", ""
	switch {
	// numbers then modifiers/extras
	case stringOnlyContainsNumbersAndSpaces(lines[0]) &&
		stringOnlyContainsMelodyModifiersAndExtras(lines[1]):
		melodyNums, lower = lines[0], lines[1]

	// modifiers/extras, then numbers, then modifiers/extras
	case len(lines) >= 3 &&
		stringOnlyContainsMelodyModifiersAndExtras(lines[0]) &&
		stringOnlyContainsNumbersAndSpaces(lines[1]) &&
		stringOnlyContainsMelodyModifiersAndExtras(lines[2]):
		upper, melodyNums, lower = lines[0], lines[1], lines[2]

	// modifiers/extras then numbers then either not modfiers or no third line
	case stringOnlyContainsMelodyModifiersAndExtras(lines[0]) &&
		stringOnlyContainsNumbersAndSpaces(lines[1]) &&
		((len(lines) >= 3 && !stringOnlyContainsMelodyModifiersAndExtras(lines[2])) ||
			len(lines) == 2):
		upper, melodyNums = lines[0], lines[1]
	default:
		return lines, elem, fmt.Errorf("could not determine melody number line and modifier line")
	}

	var msOut melodies
	melodiesFound := false
	for i, r := range melodyNums {
		if !(unicode.IsSpace(r) || unicode.IsNumber(r)) {
			return lines, elem, fmt.Errorf(
				"melodies line contains something other"+
					"than numbers and spaces (rune: %v, col: %v)", r, i)
		}
		if unicode.IsSpace(r) {
			msOut = append(msOut, melody{num: ' '})
			continue
		}

		m := melody{num: r, modifier: ' ', extra: ' '}
		chAbove, chBelow := ' ', ' '
		if len(upper) > i && !unicode.IsSpace(rune(upper[i])) {
			chAbove = rune(upper[i])
		}
		if len(lower) > i && !unicode.IsSpace(rune(lower[i])) {
			chBelow = rune(lower[i])
		}
		if runeIsExtra(chAbove) {
			m.extra = chAbove
		}
		if runeIsExtra(chBelow) {
			m.extra = chBelow
		}
		if runeIsMod(chAbove) {
			m.modifierIsAboveNum = true
			m.modifier = chAbove
		} else if runeIsMod(chBelow) {
			m.modifierIsAboveNum = false
			m.modifier = chBelow
		}

		// ensure that the melody modifier has a valid rune
		if unicode.IsSpace(m.modifier) {
			return lines, elem, fmt.Errorf("no melody modifier for the melody")
		}
		if !runeIsMod(m.modifier) {
			return lines, elem, fmt.Errorf(
				"bad modifier not '%s', '%s', or '%s' (have %v)",
				mod1, mod2, mod3, m.modifier)
		}

		msOut = append(msOut, m)
		melodiesFound = true
	}

	if !melodiesFound {
		return lines, elem, fmt.Errorf("no melodies found")
	}

	return lines[3:], msOut, nil
}

func (ms melodies) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {

	// accumulate all the used height as it's used
	usedHeight := 0.0

	// lyric font info
	fontH := GetFontHeight(lyricFontPt)
	fontW := GetCourierFontWidthFromHeight(fontH)
	xLyricStart := bnd.left - fontW/2 // - because of slight right shift in sine annotations

	// print the melodies
	melodyFontPt := lyricFontPt
	pdf.SetFont("courier", "", melodyFontPt)
	melodyFontH := GetFontHeight(melodyFontPt)
	melodyHPadding := melodyFontH * 0.3
	melodyWPadding := 0.0
	usedHeight += melodyFontH + melodyHPadding

	y := bnd.top + melodyFontH + melodyHPadding

	for i, m := range ms {

		x := xLyricStart + float64(i)*fontW + melodyWPadding

		// get the next melody number
		nextMelodyNum := ' '
		j := i + 1
		for ; j < len(ms); j++ {
			nmn := ms[j].num
			if unicode.IsNumber(nmn) {
				nextMelodyNum = nmn
				break
			}
		}
		xNext := xLyricStart + float64(j)*fontW + melodyWPadding

		m.print(pdf, x, y, fontH, nextMelodyNum, xNext)
	}

	// padding for below the melody
	usedHeight += melodyHPadding

	return bounds{bnd.top + usedHeight, bnd.left, bnd.bottom, bnd.right}
}
