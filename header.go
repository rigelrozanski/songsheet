package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

type headerContentFilled struct {
	title          string
	titleLine2     string
	date           string
	tuningTopLeft  string
	tuningTopMid   string
	tuningTopRight string
	tuningBotLeft  string
	tuningBotMid   string
	tuningBotRight string
	capo           string
	bpm            string
	timesigTop     string
	timesigBottom  string
}

func parseHeader(lines []string) (reduced []string, hc headerContentFilled, err error) {
	if len(lines) < 4 {
		return lines, hc, fmt.Errorf("improper number of "+
			"input lines, want at least 2 have %v", len(lines))
	}

	splt := strings.SplitN(lines[0], "DATE:", 2)
	if len(splt) < 2 {
		return lines, hc, errors.New("must include DATE (in first line)")
	}
	hc.title = strings.TrimRight(splt[0], " ")
	hc.date = splt[1]

	splt2 := strings.SplitN(lines[1], "|", 2)
	if len(splt2) == 2 {
		hc.titleLine2 = strings.TrimRight(splt2[0], " ")
	}

	datePos := len(splt[0])
	hc.timesigTop = string(lines[1][datePos])
	hc.timesigBottom = string(lines[2][datePos])
	hc.bpm = string(lines[1][datePos+2 : datePos+5])
	hc.capo = string(lines[2][datePos+8 : datePos+10])

	// get tuning keys
	hc.tuningTopLeft = string(lines[1][datePos+11 : datePos+13])
	hc.tuningBotLeft = string(lines[2][datePos+11 : datePos+13])
	hc.tuningTopMid = string(lines[1][datePos+13 : datePos+15])
	hc.tuningBotMid = string(lines[2][datePos+13 : datePos+15])
	hc.tuningTopRight = string(lines[1][datePos+15 : datePos+17])
	hc.tuningBotRight = string(lines[2][datePos+15 : datePos+17])

	return lines[4:], hc, nil
}

func printHeaderFilled(pdf *gofpdf.Fpdf, bnd bounds, hc *headerContentFilled) (reducedBounds bounds) {
	dateRightOffset := 2.3

	// flip string orientation if called for
	if mirrorStringsOrderFlag {
		thicknessesRev := make([]float64, len(thicknesses))
		j := len(thicknesses) - 1
		for i := 0; i < len(thicknesses); i++ {
			thicknessesRev[j] = thicknesses[i]
			j--
		}
		thicknesses = thicknessesRev
	}

	// print date
	pdf.SetFont("courier", "", 14)
	fontH := GetFontHeight(14)
	fontW := GetCourierFontWidthFromHeight(fontH)
	pdf.Text(bnd.right-dateRightOffset, bnd.top+padding-0.5*fontH, "DATE:"+hc.date)

	pdf.Text(bnd.right-dateRightOffset, bnd.top+padding+1.3*fontH, hc.timesigTop)
	pdf.Text(bnd.right-dateRightOffset, bnd.top+padding+2.5*fontH, hc.timesigBottom)
	frX1 := bnd.right - dateRightOffset
	frX2 := frX1 + fontW
	frY := bnd.top + padding + 1.5*fontH
	pdf.SetLineWidth(thinLW)
	pdf.Line(frX1, frY, frX2, frY) // fraction line

	pdf.Text(bnd.right-dateRightOffset, bnd.top+padding+1.3*fontH, "  "+hc.bpm)
	pdf.Text(bnd.right-dateRightOffset, bnd.top+padding+2.5*fontH, "  BPM")

	// print capo
	pdf.Text(bnd.right-dateRightOffset, bnd.top+padding+2.5*fontH, "       "+hc.capo)
	pdf.SetLineWidth(thickerLW)
	x1 := bnd.right - dateRightOffset + 6*fontW
	x15 := bnd.right - dateRightOffset + 6.25*fontW
	x2 := bnd.right - dateRightOffset + 6.5*fontW
	x3 := bnd.right - dateRightOffset + 6.75*fontW
	x4 := bnd.right - dateRightOffset + 8*fontW
	x5 := bnd.right - dateRightOffset + 7.5*fontW
	y1 := bnd.top + padding + 0.40*fontH
	y2 := bnd.top + padding + 1.35*fontH
	y3 := bnd.top + padding + 2.20*fontH
	y4 := bnd.top + padding + 2.50*fontH
	pdf.Line(x1, y2, x15, y3) // under gripper
	pdf.Line(x1, y2, x5, y1)  // bottom release
	pdf.SetLineWidth(1.5 * thickerLW)
	pdf.Line(x1, y2, x4, y2) // front release
	pdf.Line(x3, y2, x2, y4) // string presser

	// print guitar head
	xStringsStart := bnd.right - dateRightOffset + 9.5*fontW
	xNeckStart := xStringsStart + 1.0*fontW
	xHeadStart := xNeckStart + 1.0*fontW
	xHeadDimple := xHeadStart + 3.7*fontW
	xHeadEnd := xHeadStart + 4*fontW
	yHeadTop := bnd.top + padding + 0.5*fontH
	yNeckThinTop := bnd.top + padding + 1.0*fontH
	yNeckThinBot := bnd.top + padding + 2.0*fontH
	yHeadBot := bnd.top + padding + 2.5*fontH
	yHeadDimple := (yHeadTop + yHeadBot) / 2
	thick, thin := 0.020, 0.005

	// strings
	for i := 0; i <= 5; i++ {
		pdf.SetLineWidth((5-float64(i))/5*thick + float64(i)/5*thin)
		y := (5-float64(i))/5*yNeckThinTop + float64(i)/5*yNeckThinBot
		pdf.Line(xStringsStart, y, xNeckStart, y)
	}

	pdf.SetLineWidth(0.01)
	pdf.Line(xNeckStart, yNeckThinTop, xHeadStart, yHeadTop) // neck
	pdf.Line(xNeckStart, yNeckThinBot, xHeadStart, yHeadBot) //
	pdf.Line(xHeadStart, yHeadTop, xHeadEnd, yHeadTop)       // head
	pdf.Line(xHeadStart, yHeadBot, xHeadEnd, yHeadBot)       //
	pdf.Line(xHeadEnd, yHeadTop, xHeadDimple, yHeadDimple)   // head dimple
	pdf.Line(xHeadEnd, yHeadBot, xHeadDimple, yHeadDimple)   //

	// keys
	keyXStart := xHeadStart + 0.5*fontW
	keyXEnd := xHeadDimple - 0.5*fontW
	keyXPos := []float64{
		keyXStart,
		(keyXStart + keyXEnd) / 2,
		keyXEnd,
	}
	keyW := fontW / 2
	keyH := fontH / 2
	for i := 0; i < 3; i++ {
		x := keyXPos[i]

		//upper key
		pdf.Line(x, yHeadTop, x-keyW/2, yHeadTop-keyH)
		pdf.Line(x, yHeadTop, x+keyW/2, yHeadTop-keyH)
		pdf.Line(x-keyW/2, yHeadTop-keyH, x+keyW/2, yHeadTop-keyH)

		//lower key
		pdf.Line(x, yHeadBot, x-keyW/2, yHeadBot+keyH)
		pdf.Line(x, yHeadBot, x+keyW/2, yHeadBot+keyH)
		pdf.Line(x-keyW/2, yHeadBot+keyH, x+keyW/2, yHeadBot+keyH)
	}

	// tuning information
	pdf.SetFont("courier", "", 9)
	tuningFontH := GetFontHeight(9)
	tuningFontW := GetCourierFontWidthFromHeight(tuningFontH)

	pdf.Text(keyXPos[0]-1.5*tuningFontW, yHeadTop+1.25*tuningFontH, hc.tuningTopLeft)
	pdf.Text(keyXPos[1]-1.5*tuningFontW, yHeadTop+1.25*tuningFontH, hc.tuningTopMid)
	pdf.Text(keyXPos[2]-1.5*tuningFontW, yHeadTop+1.25*tuningFontH, hc.tuningTopRight)
	pdf.Text(keyXPos[0]-1.5*tuningFontW, yHeadBot-0.45*tuningFontH, hc.tuningBotLeft)
	pdf.Text(keyXPos[1]-1.5*tuningFontW, yHeadBot-0.45*tuningFontH, hc.tuningBotMid)
	pdf.Text(keyXPos[2]-1.5*tuningFontW, yHeadBot-0.45*tuningFontH, hc.tuningBotRight)

	////////////////////////
	// print title
	// determine title font
	titleFont := 40.0
	titleFontH, titleFontW, usedHeight := 0.0, 0.0, 0.0
	availableWidth := bnd.right - dateRightOffset - bnd.left - padding/2
	availableHeight := (yHeadBot + keyH) - (bnd.top + padding/2)
	for {
		titleFontH = 1.1 * GetFontHeight(titleFont)
		titleFontW = GetCourierFontWidthFromHeight(titleFontH)
		usedWidth1 := float64(len(hc.title)) * titleFontW
		usedWidth2 := float64(len(hc.titleLine2)) * titleFontW
		usedHeight = titleFontH
		if len(hc.titleLine2) > 0 {
			usedHeight += usedHeight
		}
		if usedWidth1 > availableWidth ||
			usedWidth2 > availableWidth ||
			usedHeight > availableHeight {
			titleFont -= 1
			continue
		}
		break
	}

	pdf.SetFont("courier", "", titleFont)
	excess := availableHeight - usedHeight
	if len(hc.titleLine2) == 0 {
		pdf.Text(bnd.left, bnd.top+usedHeight+excess/2, hc.title)
	} else {
		pdf.Text(bnd.left, bnd.top+titleFontH+excess/2, hc.title)
		pdf.Text(bnd.left, bnd.top+2*titleFontH+excess/2, hc.titleLine2)
	}

	return bounds{yHeadBot + keyH + padding, bnd.left, bnd.bottom, bnd.right}
}
