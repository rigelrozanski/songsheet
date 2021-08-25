package main

import (
	"fmt"
	"unicode"
)

type melodies []melody

var _ tssElement = melodies{}

type melody struct {
	blank              bool // no melody here, this is just a placeholder
	num                rune
	modifierIsAboveNum bool // otherwise below
	modifier           rune // either '.', '-', or '~'

	// whether to display:
	//  '(' = the number with tight brackets
	//  '/' = to add a modfier for a slide up
	//  '\' = to add a modifier for a slide down
	extra rune
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
			msOut = append(msOut, melody{blank: true})
			continue
		}

		m := melody{blank: false, num: r, modifier: ' ', extra: ' '}
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
	melodyFontW := GetCourierFontWidthFromHeight(melodyFontH)
	melodyHPadding := melodyFontH * 0.3
	melodyWPadding := 0.0
	yNum := bnd.top + melodyFontH + melodyHPadding*2
	usedHeight += melodyFontH + melodyHPadding*3
	for i, melody := range ms {
		if melody.blank {
			continue
		}

		// print number
		xNum := xLyricStart + float64(i)*fontW + melodyWPadding
		pdf.Text(xNum, yNum, string(melody.num))

		// print modifier
		switch melody.modifier {
		case mod1:
			xMod := xNum + melodyFontW/2
			yMod := yNum + melodyHPadding*1.5
			if melody.modifierIsAboveNum {
				yMod = yNum - melodyFontH - melodyHPadding/1.5
			}
			pdf.Circle(xMod, yMod, melodyHPadding/1.5, "F")
		case mod2:
			xModStart := xNum
			xModEnd := xNum + melodyFontW
			yMod := yNum + melodyHPadding
			if melody.modifierIsAboveNum {
				yMod = yNum - melodyFontH - melodyHPadding
			}
			pdf.SetLineWidth(thinishLW)
			pdf.Line(xModStart, yMod, xModEnd, yMod)
		case mod3:
			xModStart := xNum
			xModMid := xNum + melodyFontW/2
			xModEnd := xNum + melodyFontW
			yMod := yNum + melodyHPadding/2
			yModMid := yMod + melodyHPadding*2
			if melody.modifierIsAboveNum {
				yMod = yNum - melodyFontH - melodyHPadding/2
				yModMid = yMod - melodyHPadding*2
			}
			pdf.SetLineWidth(thinishLW)
			pdf.Curve(xModStart, yMod, xModMid, yModMid, xModEnd, yMod, "")
		default:
			panic(fmt.Errorf("unknown modifier %v", melody.modifier))
		}

		// print extra decorations

		switch melody.extra {
		case extraBrac:
			xBrac1 := xNum - fontW*0.5
			xBrac2 := xNum + fontW*0.5
			yBrac := yNum - melodyHPadding/2 // shift for looks
			pdf.Text(xBrac1, yBrac, "(")
			pdf.Text(xBrac2, yBrac, ")")

		case extraSldUp:

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
			switch melody.num {
			case '0':
				xSldStart = xNum + fontW*XStart0
				ySldStart = yNum - melodyHPadding*YStart0
			case '1':
				xSldStart = xNum + fontW*XStart1
				ySldStart = yNum - melodyHPadding*YStart1
			case '2':
				xSldStart = xNum + fontW*XStart2
				ySldStart = yNum - melodyHPadding*YStart2
			case '3':
				xSldStart = xNum + fontW*XStart3
				ySldStart = yNum - melodyHPadding*YStart3
			case '4':
				xSldStart = xNum + fontW*XStart4
				ySldStart = yNum - melodyHPadding*YStart4
			case '5':
				xSldStart = xNum + fontW*XStart5
				ySldStart = yNum - melodyHPadding*YStart5
			case '6':
				xSldStart = xNum + fontW*XStart6
				ySldStart = yNum - melodyHPadding*YStart6
			case '7':
				xSldStart = xNum + fontW*XStart7
				ySldStart = yNum - melodyHPadding*YStart7
			case '8':
				xSldStart = xNum + fontW*XStart8
				ySldStart = yNum - melodyHPadding*YStart8
			case '9':
				xSldStart = xNum + fontW*XStart9
				ySldStart = yNum - melodyHPadding*YStart9
			default:
				draw = false
			}

			// determine the ending location
			xNumNext := xLyricStart + float64(j)*fontW + melodyWPadding
			xSldEnd, ySldEnd := 0.0, 0.0
			switch nextMelodyNum {
			case '0':
				xSldEnd = xNumNext + fontW*XEnd0
				ySldEnd = yNum - fontH + melodyHPadding*YEnd0
			case '1':
				xSldEnd = xNumNext + fontW*XEnd1
				ySldEnd = yNum - fontH + melodyHPadding*YEnd1
			case '2':
				xSldEnd = xNumNext + fontW*XEnd2
				ySldEnd = yNum - fontH + melodyHPadding*YEnd2
			case '3':
				xSldEnd = xNumNext + fontW*XEnd3
				ySldEnd = yNum - fontH + melodyHPadding*YEnd3
			case '4':
				xSldEnd = xNumNext + fontW*XEnd4
				ySldEnd = yNum - fontH + melodyHPadding*YEnd4
			case '5':
				xSldEnd = xNumNext + fontW*XEnd5
				ySldEnd = yNum - fontH + melodyHPadding*YEnd5
			case '6':
				xSldEnd = xNumNext + fontW*XEnd6
				ySldEnd = yNum - fontH + melodyHPadding*YEnd6
			case '7':
				xSldEnd = xNumNext + fontW*XEnd7
				ySldEnd = yNum - fontH + melodyHPadding*YEnd7
			case '8':
				xSldEnd = xNumNext + fontW*XEnd8
				ySldEnd = yNum - fontH + melodyHPadding*YEnd8
			case '9':
				xSldEnd = xNumNext + fontW*XEnd9
				ySldEnd = yNum - fontH + melodyHPadding*YEnd9
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
			switch melody.num {
			case '0':
				xSldStart = xNum + fontW*XStart0
				ySldStart = yNum - fontH + melodyHPadding*YStart0
			case '1':
				xSldStart = xNum + fontW*XStart1
				ySldStart = yNum - fontH + melodyHPadding*YStart1
			case '2':
				xSldStart = xNum + fontW*XStart2
				ySldStart = yNum - fontH + melodyHPadding*YStart2
			case '3':
				xSldStart = xNum + fontW*XStart3
				ySldStart = yNum - fontH + melodyHPadding*YStart3
			case '4':
				xSldStart = xNum + fontW*XStart4
				ySldStart = yNum - fontH + melodyHPadding*YStart4
			case '5':
				xSldStart = xNum + fontW*XStart5
				ySldStart = yNum - fontH + melodyHPadding*YStart5
			case '6':
				xSldStart = xNum + fontW*XStart6
				ySldStart = yNum - fontH + melodyHPadding*YStart6
			case '7':
				xSldStart = xNum + fontW*XStart7
				ySldStart = yNum - fontH + melodyHPadding*YStart7
			case '8':
				xSldStart = xNum + fontW*XStart8
				ySldStart = yNum - fontH + melodyHPadding*YStart8
			case '9':
				xSldStart = xNum + fontW*XStart9
				ySldStart = yNum - fontH + melodyHPadding*YStart9
			default:
				draw = false
			}

			// determine the ending location
			xNumNext := xLyricStart + float64(j)*fontW + melodyWPadding
			xSldEnd, ySldEnd := 0.0, 0.0
			switch nextMelodyNum {
			case '0':
				xSldEnd = xNumNext + fontW*XEnd0
				ySldEnd = yNum - melodyHPadding*YEnd0
			case '1':
				xSldEnd = xNumNext + fontW*XEnd1
				ySldEnd = yNum - melodyHPadding*YEnd1
			case '2':
				xSldEnd = xNumNext + fontW*XEnd2
				ySldEnd = yNum - melodyHPadding*YEnd2
			case '3':
				xSldEnd = xNumNext + fontW*XEnd3
				ySldEnd = yNum - melodyHPadding*YEnd3
			case '4':
				xSldEnd = xNumNext + fontW*XEnd4
				ySldEnd = yNum - melodyHPadding*YEnd4
			case '5':
				xSldEnd = xNumNext + fontW*XEnd5
				ySldEnd = yNum - melodyHPadding*YEnd5
			case '6':
				xSldEnd = xNumNext + fontW*XEnd6
				ySldEnd = yNum - melodyHPadding*YEnd6
			case '7':
				xSldEnd = xNumNext + fontW*XEnd7
				ySldEnd = yNum - melodyHPadding*YEnd7
			case '8':
				xSldEnd = xNumNext + fontW*XEnd8
				ySldEnd = yNum - melodyHPadding*YEnd8
			case '9':
				xSldEnd = xNumNext + fontW*XEnd9
				ySldEnd = yNum - melodyHPadding*YEnd9
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
	// the greatest use of height is from the midpoint of the mod3 modifier
	usedHeight += melodyHPadding/2 + melodyHPadding*3

	return bounds{bnd.top + usedHeight, bnd.left, bnd.bottom, bnd.right}
}
