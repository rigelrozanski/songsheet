package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/rigelrozanski/thranch/quac"
	"github.com/spf13/cobra"
)

var (
	GenerateCmd = &cobra.Command{
		Use:   "gen [qu-id]",
		Short: "generate the pdf of the songsheet at the qu-id",
		Args:  cobra.ExactArgs(1),
		RunE:  genCmd,
	}

	lyricFontPt  float64
	longestHumps float64

	printTitleFlag         bool
	spacingRatioFlag       float64
	sineAmplitudeRatioFlag float64
	numColumnsFlag         uint16

	subsupSizeMul = 0.65 // size of sub and superscript relative to thier root's size
)

func init() {
	GenerateCmd.PersistentFlags().BoolVar(
		&mirrorStringsOrderFlag, "mirror", false,
		"mirror string positions")
	GenerateCmd.PersistentFlags().Uint16Var(
		&numColumnsFlag, "columns", 2,
		"number of columns to print song into")
	GenerateCmd.PersistentFlags().Float64Var(
		&spacingRatioFlag, "spacing-ratio", 1.5,
		"ratio of the spacing to the lyric-lines")
	GenerateCmd.PersistentFlags().Float64Var(
		&sineAmplitudeRatioFlag, "amp-ratio", 0.8,
		"ratio of amplitude of the sine curve to the lyric text")
	RootCmd.AddCommand(GenerateCmd)
}

func genCmd(cmd *cobra.Command, args []string) error {

	pdf := gofpdf.New("P", "in", "Letter", "")
	pdf.SetMargins(0, 0, 0)
	pdf.AddPage()

	// each line of text from the input file
	// is attempted to be fit into elements
	// in the order provided within elemKinds
	elemKinds := []tssElement{
		spacer{},
		chordChart{},
		sine{},
		melodies{},
		lyrics{},
	}

	if numColumnsFlag < 1 {
		return errors.New("numColumnsFlag must be greater than 1")
	}

	quid, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	content, found := quac.GetContentByID(uint32(quid))
	if !found {
		return fmt.Errorf("could not find anything under id: %v", quid)
	}
	lines := strings.Split(string(content), "\n")
	lines = deleteComments(lines)

	// get the header
	lines, hc, err := parseHeader(lines)
	filename := fmt.Sprintf("songsheet_%v.pdf", hc.title)

	bnd := bounds{padding, padding, 11, 8.5}
	if printTitleFlag {
		hc.title = ""
	}
	bnd = printHeaderFilled(pdf, bnd, &hc)

	//seperate out remaining bounds into columns
	bndsColsIndex := 0
	bndsCols := splitBoundsIntoColumns(bnd, numColumnsFlag)
	if len(bndsCols) == 0 {
		panic("no bound columns")
	}

	//determine lyricFontPt
	longestHumps, lyricFontPt, err = determineLyricFontPt(lines, bndsCols[0])
	if err != nil {
		return err
	}

	// get contents of songsheet
	// parse all the elems
	parsedElems := []tssElement{}

OUTER:
	if len(lines) > 0 {
		allErrs := []string{}
		for _, elem := range elemKinds {
			reduced, newElem, err := elem.parseText(lines)
			if err == nil {
				lines = reduced
				parsedElems = append(parsedElems, newElem)
				goto OUTER
			} else {
				allErrs = append(allErrs, err.Error())
			}
		}
		return fmt.Errorf("could not parse song at line %+v\n all errors%+v\n", lines, allErrs)
	}

	// print the songsheet elements
	//  - use a dummy pdf to test whether the borders are exceeded within
	//    the current column, if so move to the next column
	for _, el := range parsedElems {
		dummy := dummyPdf{}
		bndNew := el.printPDF(dummy, bndsCols[bndsColsIndex])
		if bndNew.Height() < padding/2 {
			bndsColsIndex++
			if bndsColsIndex >= len(bndsCols) {
				return errors.New("song doesn't fit on one sheet " +
					"(functionality not built yet for multiple sheets)") // TODO
			}
		}
		bndsCols[bndsColsIndex] = el.printPDF(pdf, bndsCols[bndsColsIndex])
	}

	return pdf.OutputFileAndClose(filename)
}

func splitBoundsIntoColumns(bnd bounds, numCols uint16) (splitBnds []bounds) {
	width := (bnd.right - bnd.left) / float64(numCols)
	for i := uint16(0); i < numCols; i++ {
		b := bounds{
			top:    bnd.top,
			bottom: bnd.bottom,
			left:   bnd.left + float64(i)*width,
			right:  bnd.left + float64(i+1)*width,
		}
		splitBnds = append(splitBnds, b)
	}
	return splitBnds
}
