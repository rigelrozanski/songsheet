package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jung-kurt/gofpdf"
)

// TODO
// - create an amplitude decay factor (flag) allow for decays
//   to happen in the middle of sine
//    - also allow for pauses (no sine at all)
// - use of sine instead of cos with different text hump pattern:   _
//                                                                 / \_

type sine struct {
	hasPlaybackTime bool
	pt              playbackTime
	ptCharPosition  int // number of humps to the playback position
	humps           float64
	trailingHumps   float64 // the sine curve reduces its amplitude to zero during these
	alongAxis       []sineAnnotation
	alongSine       []sineAnnotation
}

func (sas sine) totalHumps() float64 {
	return sas.humps + sas.trailingHumps
}

type sineAnnotation struct {
	position    float64 // in humps
	bolded      bool    // whether the whole unit is bolded
	ch          rune    // main character
	subscript   rune    // following subscript character
	superscript rune    // following superscript character
	slide       bool    // annotation calls for a slide AFTER the ch
	isMelody    bool
	mel         melody
}

// print the sineAnnotation at the provided location
func (sa sineAnnotation) printAlongAxis(pdf Pdf, x, y float64, fontH float64) {

	fontPt := GetFontPt(fontH)
	fontW := GetCourierFontWidthFromHeight(fontH)
	XsubsupCrunch := fontW * 0.1 // squeeze the sub and super script into the chord a bit
	fontHSubSup := fontH * subsupSizeMul
	fontPtSubSup := GetFontPt(fontHSubSup)

	bolded := ""
	if sa.bolded {
		bolded = "B"
	}
	pdf.SetFont("courier", bolded, fontPt)

	if sa.isMelody {
		x += fontW * 0.16 // weird corrections to make the
		y -= fontH * 0.15 // numbers feel in the middle of the sine
		sa.mel.print(pdf, x, y, fontH, ' ', 0)
		return
	}

	pdf.Text(x, y, string(sa.ch))

	// print sub or super script if exists
	if sa.subscript != ' ' || sa.superscript != ' ' {
		Xsubsup := x + fontW - XsubsupCrunch
		pdf.SetFont("courier", bolded, fontPtSubSup)
		if sa.subscript != ' ' {
			Ysub := y - fontH/2 + fontHSubSup
			pdf.Text(Xsubsup, Ysub, string(sa.subscript))
		}
		if sa.superscript != ' ' {
			Ysuper := y - fontH/2
			pdf.Text(Xsubsup, Ysuper, string(sa.superscript))
		}
	}

}

// NewsineAnnotation creates a new sineAnnotation object
func NewSineAnnotation(position float64, bolded bool,
	ch, subscript, superscript rune, slide bool) sineAnnotation {
	return sineAnnotation{
		position:    position,
		bolded:      bolded,
		ch:          ch,
		subscript:   subscript,
		superscript: superscript,
		slide:       slide,
	}
}

var _ tssElement = sine{}

func GetSASFromTopLines(lines []string) (sas sine, err error) {

	// the annotated sine must come in 4 OR 5 Lines
	//    ex.   desciption
	// 1) F              along axis annotations
	// 2) _   _   _   _  text representation of the sine humps (top)
	// 3)  \_/ \_/ \_/   text representation of the sine humps (bottom)
	// 4)   ^   ^ 1   v  annotations along the sine curve
	// 5)     00:03.14   (optional) playback time position

	if len(lines) < 4 {
		return sas, fmt.Errorf("improper number of input lines,"+
			"want 4 have %v", len(lines))
	}

	// ensure that the second and third lines start with at least 1 sine hump
	//_
	// \_/
	if !(strings.HasPrefix(lines[1], "_") && strings.HasPrefix(lines[2], " \\_/")) {
		return sas, fmt.Errorf("first lines are not sine humps")
	}

	// get the playback time if it exists
	if len(lines) > 4 {
		pt, ptCharPosition, ptFound := getPlaybackTimeFromLine(lines[4])
		sas = sine{
			hasPlaybackTime: ptFound,
			ptCharPosition:  ptCharPosition,
			pt:              pt,
		}
	}

	return sas, nil
}

type playbackTime struct {
	// string representation
	//   mn:se.cs
	// where
	//   mn = minutes
	//   se = seconds
	//   cs = centi-seconds (1/100th of a second)
	str string // string representation
	t   time.Time
}

func (pt playbackTime) AddDur(d time.Duration) (ptOut playbackTime) {
	ptOut.t = pt.t.Add(d)
	newDur := ptOut.t.Sub(time.Time{})

	minsFl := math.Trunc(newDur.Seconds() / 60)
	minsStr := fmt.Sprintf("%02v", minsFl)
	secsFl := newDur.Seconds() - 60*minsFl
	secsStr := fmt.Sprintf("%05v", strconv.FormatFloat(secsFl, 'f', 2, 64))
	ptOut.str = fmt.Sprintf("%v:%v", minsStr, secsStr) // secsStr contains two decimals
	return ptOut
}

// 00:00.00
func getPlaybackTimeFromLine(line string) (pt playbackTime, ptCharPosition int, found bool) {
	tr := strings.TrimSpace(line)
	if len(tr) != 8 {
		return pt, 0, false
	}
	str := tr
	spl1 := strings.SplitN(tr, ":", 2)
	if len(spl1) != 2 {
		return pt, 0, false
	}
	spl2 := strings.SplitN(spl1[1], ".", 2)
	if len(spl1) != 2 {
		return pt, 0, false
	}

	mins, err := strconv.Atoi(spl1[0])
	if err != nil {
		return pt, 0, false
	}
	secs, err := strconv.Atoi(spl2[0])
	if err != nil {
		return pt, 0, false
	}
	centiSecs, err := strconv.Atoi(spl2[1])
	if err != nil {
		return pt, 0, false
	}

	// get the time in the golang time format
	dur := time.Minute * time.Duration(mins)
	dur += time.Second * time.Duration(secs)
	dur += time.Millisecond * 10 * time.Duration(centiSecs)
	t := time.Time{}.Add(dur)

	pt = playbackTime{
		str: str,
		t:   t,
	}

	ptCharPosition = len(line) - len(strings.TrimLeft(line, " "))
	return pt, ptCharPosition, true
}

func (s sine) parseText(lines []string) (reduced []string, elem tssElement, err error) {

	sas, err := GetSASFromTopLines(lines)
	if err != nil {
		return lines, elem, err
	}

	humpsChars := len(strings.TrimSpace(lines[1]))
	secondLineTrimTrail := strings.TrimRight(lines[2], ".")
	// +1 for the leading space just trimmed
	secondLineLen := len(strings.TrimSpace(secondLineTrimTrail)) + 1
	if humpsChars < secondLineLen {
		humpsChars = secondLineLen
	}
	humps := float64(humpsChars) / charsToaHump

	trailingHumpsChars := strings.Count(lines[2], ".")
	trailingHumps := float64(trailingHumpsChars) / charsToaHump

	// parse along axis text
	alongAxis := []sineAnnotation{}
	fl := lines[0]
	for pos := 0; pos < len(fl); pos++ {
		ch := rune(fl[pos])
		if ch == ' ' {
			continue
		}
		bolded := false

		hasNextCh := pos+1 < len(fl)
		hasNextNextCh := pos+2 < len(fl)
		hasNextNextNextCh := pos+3 < len(fl)
		nextCh, nextNextCh, nextNextNextCh := ' ', ' ', ' '
		if hasNextCh {
			nextCh = rune(fl[pos+1])
		}
		if hasNextNextCh {
			nextNextCh = rune(fl[pos+2])
		}
		if hasNextNextNextCh {
			nextNextNextCh = rune(fl[pos+3])
		}

		// check if it's a melody
		if hasNextCh {
			mel, success := NewMelodyFromTwoChars(ch, nextCh)
			if success {
				sa := sineAnnotation{position: float64(pos) / 4, isMelody: true, mel: mel}
				alongAxis = append(alongAxis, sa)
				pos++
				continue
			}
		}

		if unicode.IsLetter(ch) &&
			unicode.IsUpper(ch) {

			bolded = true
		}

		subscript, superscript, slide := determineChordsSubscriptSuperscriptSlide(
			ch, nextCh, nextNextCh, nextNextNextCh)

		alongAxis = append(alongAxis,
			NewSineAnnotation(float64(pos)/4, bolded, ch,
				subscript, superscript, slide))

		// sub or superscripts mean that we've already used up the next
		// characters hence we can advance faster than the for def
		if subscript != ' ' {
			pos++
		}
		if superscript != ' ' {
			pos++
		}
		if slide {
			pos++
		}
	}

	// parse along sine text
	alongSine := []sineAnnotation{}
	for pos, ch := range lines[3] {
		if ch == ' ' {
			continue
		}

		bolded := false
		if ch == 'V' {
			ch = 'v'
			bolded = true
		}
		if ch == 'A' {
			ch = '^'
			bolded = true
		}

		alongSine = append(alongSine,
			NewSineAnnotation(float64(pos)/4, bolded, ch, ' ', ' ', false))
	}

	sas.humps = humps
	sas.trailingHumps = trailingHumps
	sas.alongAxis = alongAxis
	sas.alongSine = alongSine
	if sas.hasPlaybackTime {
		return lines[5:], sas, nil
	}
	return lines[4:], sas, nil
}

func (s sine) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {

	// Print the sine function
	pdf.SetLineWidth(thinLW)
	resolution := 0.01
	lfh := GetFontHeight(lyricFontPt)
	amplitude := sineAmplitudeRatioFlag * lfh
	chhbs := lfh / 3      // char height beyond sine
	tipHover := chhbs / 2 // char hover when on the sine tip

	usedHeight := 2 * ( // times 2 because both sides of the sine
	amplitude +         // for the sine curve
		chhbs + // for the text extending out of the sine curve
		tipHover) // for the floating text extendion out of the sine tips

	xStart := bnd.left
	xEnd := bnd.right - padding
	width := xEnd - xStart
	trailingWidth := 0.0
	if s.humps < longestHumps {
		trailingWidth = width * s.trailingHumps / longestHumps
		width = width * s.humps / longestHumps
	}
	frequency := math.Pi * 2 * s.humps / width
	yStart := bnd.top + usedHeight/2
	lastPointX := xStart
	lastPointY := yStart
	pdf.SetLineWidth(thinestLW)

	// regular sinepart
	eqX := 0.0
	for ; true; eqX += resolution {
		if eqX > width {
			break
		}
		eqY := amplitude * math.Cos(frequency*eqX)

		if eqX > 0 {

			// -eqY because starts from topleft corner
			pdf.Line(lastPointX, lastPointY, xStart+eqX, yStart-eqY)
		}
		lastPointX = xStart + eqX
		lastPointY = yStart - eqY
	}

	// trailing sine part
	maxWidth := width + trailingWidth
	for ; true; eqX += resolution {
		if eqX > maxWidth {
			break
		}

		// trailing amplitude
		ta := amplitude * (maxWidth - eqX) / trailingWidth

		eqY := ta * math.Cos(frequency*eqX)

		if eqX > 0 {
			// -eqY because starts from topleft corner
			pdf.Line(lastPointX, lastPointY, xStart+eqX, yStart-eqY)
		}
		lastPointX = xStart + eqX
		lastPointY = yStart - eqY
	}

	///////////////
	// print the text along axis

	// (max multiplier would be 2 as the text is
	// centered between the positive and neg amplitude)
	fontH := amplitude * 1.7
	fontW := GetCourierFontWidthFromHeight(fontH)

	for i, aa := range s.alongAxis {
		X := xStart + (aa.position/s.humps)*width - fontW/2
		Y := yStart + fontH/2 // so the text is centered along the sine axis
		aa.printAlongAxis(pdf, X, Y, fontH)

		// print slide
		if aa.slide && i+1 < len(s.alongAxis) {
			posStep := 0.05
			posStart := aa.position
			posEnd := s.alongAxis[i+1].position
			posSteps := (posEnd - posStart) / posStep

			alphaStart := 0.07
			alphaEnd := 0.0
			alphaStep := (alphaEnd - alphaStart) / posSteps

			alpha := alphaStart
			for p := posStart; p < posEnd; p += posStep {
				if alpha < 0 {
					alpha = 0 // can happen due to rounding errors
				}
				pdf.SetAlpha(alpha, "")
				X := xStart + (p/s.humps)*width - fontW/2
				aa.printAlongAxis(pdf, X, Y, fontH)

				alpha += alphaStep
			}
			pdf.SetAlpha(1.0, "")
		}
	}

	// print the characters along the sine curve
	pdf.SetLineCapStyle("square")
	defer pdf.SetLineCapStyle("")
	for _, as := range s.alongSine {
		if as.ch == ' ' {
			continue
		}

		// determine hump position
		eqX := (as.position / s.humps) * width
		eqY := amplitude * math.Cos(frequency*eqX)

		// determine bold params
		bolded := ""
		if as.bolded {
			pdf.SetLineWidth(thickerLW)
			bolded = "B"
		} else {
			pdf.SetLineWidth(thinishLW)
		}

		// move the character if it intersects with
		// on of the characters along the axis

		rem := as.position - math.Trunc(as.position)
		if rem == 0.25 || rem == 0.75 {
			for _, aa := range s.alongAxis {
				if as.position == aa.position {
					eqY = amplitude // send it to the top
					break
				}
				if as.position == aa.position+0.25 && (aa.subscript != ' ' || aa.superscript != ' ') {
					eqY = amplitude // send it to the top
					break
				}
			}
		}

		// character height which extends beyond the sine curve
		switch as.ch {
		case 'v':
			tipX := xStart + eqX
			tipY := yStart - eqY
			dec := (as.position) - math.Trunc(as.position)
			if dec == 0 || dec == 0.5 {
				tipY -= tipHover
			}
			// 45deg angles to the tip
			if as.bolded { // draw a closed polygon instead of just lines
				pts := []gofpdf.PointType{
					{tipX - chhbs, tipY - chhbs},
					{tipX + chhbs, tipY - chhbs},
					{tipX, tipY},
				}
				pdf.Polygon(pts, "FD")
			} else {
				pdf.Line(tipX-chhbs, tipY-chhbs, tipX, tipY)
				pdf.Line(tipX, tipY, tipX+chhbs, tipY-chhbs)
			}
		case '^':
			tipX := xStart + eqX
			tipY := yStart - eqY
			dec := (as.position) - math.Trunc(as.position)
			if dec == 0 || dec == 0.5 {
				tipY += tipHover
			}
			// 45deg angles to the tip

			if as.bolded { // draw a closed polygon instead of just lines
				pts := []gofpdf.PointType{
					{tipX - chhbs, tipY + chhbs},
					{tipX + chhbs, tipY + chhbs},
					{tipX, tipY},
				}
				pdf.Polygon(pts, "FD")
			} else {
				pdf.Line(tipX-chhbs, tipY+chhbs, tipX, tipY)
				pdf.Line(tipX, tipY, tipX+chhbs, tipY+chhbs)
			}
		case '|':
			x := xStart + eqX
			pdf.Line(x, yStart-amplitude-chhbs, x, yStart+amplitude+chhbs)

		default:
			h := 2 * chhbs // font height in inches
			fontPt := GetFontPt(h)
			w := GetCourierFontWidthFromHeight(h) // font width

			// we want the character to be centered about the sine curve
			pdf.SetFont("courier", bolded, fontPt)
			tipX := xStart + eqX
			tipY := yStart - eqY
			shiftH := h / 2

			// shift the character to the outside of the curve if on
			// one of the peaks/troughs
			rem := as.position - math.Trunc(as.position)
			if rem == 0.5 {
				shiftH += h / 2
			}
			if rem == 0.0 {
				shiftH -= h / 2
			}

			pdf.Text(tipX-(w/2), tipY+shiftH, string(as.ch))
		}
	}

	return bounds{bnd.top + usedHeight, bnd.left, bnd.bottom, bnd.right}
}
