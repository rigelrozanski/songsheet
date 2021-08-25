package commands

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jung-kurt/gofpdf"
	"github.com/rigelrozanski/thranch/quac"
	"github.com/spf13/cobra"
)

/*
TODO
- flags for colours!
  - about the sine curve colours
- bass line strings annotations
- make new file format (and search using qu OR in the current directory for files
  with this new type)
- break this program out to a new repo

- extras for melody
  - '_' for steadyness (streches beyond note)
  - 'v' for vibrato
  - 'V' for intense vibrato
  - '|' for halting singing
  - ability to combine '_', 'v', 'V' and '|'
*/

var (
	SongsheetFilledCmd = &cobra.Command{
		Use:   "songsheetfilled [qu-id]",
		Short: "print filled songsheet from qu id",
		Args:  cobra.ExactArgs(1),
		RunE:  songsheetFilledCmd,
	}

	SongsheetPlaybackTimeCmd = &cobra.Command{
		Use:   "songsheet-filled-playback-time [filepath] [cursor-x] [cursor-y]",
		Short: "return the playback time (mm:ss:[cs][cs]) for the current position",
		Args:  cobra.ExactArgs(3),
		RunE:  songsheetPlaybackTimeCmd,
	}

	IsSongsheetCmd = &cobra.Command{
		Use:   "is-songsheet [filepath]",
		Short: "print TRUE or FALSE if the file is a songsheet",
		Args:  cobra.ExactArgs(1),
		RunE:  isSongsheetCmd,
	}

	GetSongsheetAudioCmd = &cobra.Command{
		Use:   "songsheet-audio [filepath]",
		Short: "print the audio filepath if exists or allocate a new file if it doesn't",
		Args:  cobra.ExactArgs(1),
		RunE:  getSongsheetAudioCmd,
	}

	HasSongsheetAudioCmd = &cobra.Command{
		Use:   "songsheet-has-audio [filepath]",
		Short: "print TRUE or FALSE if the file has associated audio",
		Args:  cobra.ExactArgs(1),
		RunE:  hasSongsheetAudioCmd,
	}

	SongsheetFillBPM = &cobra.Command{
		Use:   "songsheet-fill-bpm [filepath]",
		Short: "print TRUE or FALSE if the file has associated audio",
		Args:  cobra.ExactArgs(1),
		RunE:  songsheetFillBPMCmd,
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
	quac.Initialize(os.ExpandEnv("$HOME/.thranch_config"))

	SongsheetFilledCmd.PersistentFlags().BoolVar(
		&mirrorStringsOrderFlag, "mirror", false,
		"mirror string positions")
	SongsheetFilledCmd.PersistentFlags().Uint16Var(
		&numColumnsFlag, "columns", 2,
		"number of columns to print song into")
	SongsheetFilledCmd.PersistentFlags().Float64Var(
		&spacingRatioFlag, "spacing-ratio", 1.5,
		"ratio of the spacing to the lyric-lines")
	SongsheetFilledCmd.PersistentFlags().Float64Var(
		&sineAmplitudeRatioFlag, "amp-ratio", 0.8,
		"ratio of amplitude of the sine curve to the lyric text")
	RootCmd.AddCommand(SongsheetFilledCmd)
	RootCmd.AddCommand(SongsheetPlaybackTimeCmd)
	RootCmd.AddCommand(IsSongsheetCmd)
	RootCmd.AddCommand(GetSongsheetAudioCmd)
	RootCmd.AddCommand(HasSongsheetAudioCmd)
	RootCmd.AddCommand(SongsheetFillBPM)
}

func isSongsheetCmd(cmd *cobra.Command, args []string) error {
	filepath := args[0]
	if strings.Contains(filepath, "songsheet") {
		fmt.Printf("TRUE")
		return nil
	}
	fmt.Printf("FALSE")
	return nil
}

const (
	audioLinePrefix = "// AUDIO-ID="
)

func hasSongsheetAudio(lines []string) (yesitdoes bool, audiofilepath string, err error) {
	for _, line := range lines {
		if strings.HasPrefix(line, audioLinePrefix) {
			quidStr := strings.TrimPrefix(line, audioLinePrefix)
			quid, err := strconv.Atoi(quidStr)
			if err != nil {
				return false, "", err
			}
			audiofilepath, found := quac.GetFilepathByID(uint32(quid))
			if !found {
				return false, "", nil
			}
			return true, audiofilepath, nil
		}
	}
	return false, "", nil
}

func songsheetFillBPMCmd(cmd *cobra.Command, args []string) error {
	filepath := args[0]
	if !strings.Contains(filepath, "songsheet") {
		return errors.New("not a songsheet cannot calc bpm")
	}
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")

	// get the list of all lines and sasses
	lasses := getLasses(lines)

	// convert the sasses into an array of characters
	type charPos struct {
		hasPT bool
		pt    playbackTime
	}
	charPoss := []charPos{}
	for _, s := range lasses {
		maxChars := int((s.sas.totalHumps() * charsToaHump) + 0.00001) // float rounding
		for j := 0; j < maxChars; j++ {
			cp := charPos{}
			if s.sas.hasPlaybackTime && s.sas.ptCharPosition == j {
				cp.hasPT = true
				cp.pt = s.sas.pt
			}
			charPoss = append(charPoss, cp)
		}
	}

	// get the first and last playback times
	var ptFirst, ptLast playbackTime
	ptFirstFound, ptLastFound := false, false
	unusedChars := 0
	for i := 0; i < len(charPoss); i++ {
		cp := charPoss[i]
		if cp.hasPT {
			ptFirst = cp.pt
			ptFirstFound = true
			break
		}
		unusedChars++
	}
	for i := len(charPoss) - 1; i >= 0; i-- {
		cp := charPoss[i]
		if cp.hasPT {
			ptLast = cp.pt
			ptLastFound = true
			break
		}
		unusedChars++
	}
	if !ptFirstFound || !ptLastFound {
		err := fmt.Errorf("//ERROR: couldn't find two playback times to calculate bpm with")
		content = []byte(err.Error() + "\n" + string(content))
		err = ioutil.WriteFile(filepath, content, 0666)
		return err
	}

	// calculate the bpm
	minutes := ptLast.t.Sub(ptFirst.t).Minutes()
	beats := float64(len(charPoss)-unusedChars) / charsToaHump
	bpm := beats / minutes
	if bpm < 50 { // some normalization
		bpm *= 2
	}
	if bpm > 200 { // some normalization
		bpm /= 2
	}

	// write the bpm line
	newLine := fmt.Sprintf("// BPM: %v", int(math.Round(bpm)))
	content = []byte(newLine + "\n" + string(content))
	err = ioutil.WriteFile(filepath, content, 0666)
	return nil
}

func hasSongsheetAudioCmd(cmd *cobra.Command, args []string) error {
	content, err := ioutil.ReadFile(args[0])
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	has, _, err := hasSongsheetAudio(lines)
	if err != nil {
		fmt.Printf("FALSE")
		return err
	}
	if has {
		fmt.Printf("TRUE")
	} else {
		fmt.Printf("FALSE")
	}
	return nil
}

func getSongsheetAudioCmd(cmd *cobra.Command, args []string) error {
	content, err := ioutil.ReadFile(args[0])
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")

	has, filepath, err := hasSongsheetAudio(lines)
	if err != nil {
		return err
	}
	if has {
		fmt.Printf(filepath)
		return nil
	}

	// if not found allocate a new file for this purpose
	// and add it to the songsheet
	origIdea := quac.NewIdeaFromFilename(args[0], false)
	clumpedTags := origIdea.GetClumpedTags()
	// add the original tags to this new entry
	audioFilepath, quID := quac.NewEmptyAudioEntry(clumpedTags)
	newLine := fmt.Sprintf("%v%v", audioLinePrefix, quID)
	content = []byte(newLine + "\n" + string(content))
	err = ioutil.WriteFile(args[0], content, 0666)
	if err != nil {
		return err
	}
	fmt.Printf(audioFilepath)
	return nil
}

func deleteComments(lines []string) (out []string) {
LOOP:
	for _, line := range lines {
		switch {
		// do not include this line
		case strings.HasPrefix(line, "//"):
			continue LOOP

		// take everything before the comment
		case strings.Contains(line, "//"):
			splt := strings.SplitN(line, "//", 2)
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

func getLasses(lines []string) (lasses lineAndSasses) {
	el := singleAnnotatedSine{} // dummy element to make the call
	for yI := 0; yI < len(lines); yI++ {
		workingLines := lines[yI:]
		_, sasEl, err := el.parseText(workingLines)
		if err == nil {
			sas := sasEl.(singleAnnotatedSine)
			ls := lineAndSas{int16(yI), sas}
			lasses = append(lasses, ls)
		}
	}
	return lasses
}

func songsheetPlaybackTimeCmd(cmd *cobra.Command, args []string) error {

	// get the relevant file
	content, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Printf("BAD-PLAYBACK-TIME")
		return err
	}
	lines := strings.Split(string(content), "\n")
	origLenLines := len(lines)
	lines = deleteComments(lines)
	commentLines := origLenLines - len(lines)

	curX, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Printf("BAD-PLAYBACK-TIME")
		return err
	}

	curY, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Printf("BAD-PLAYBACK-TIME")
		return err
	}
	curY -= commentLines

	// get the list of all lines and sasses
	lasses := getLasses(lines)

	// determine the sas belonging to the cursor
	curI := 0
LOOP:
	for i, s := range lasses {
		switch {
		case (curY - 1) == int(s.lineNo):
			curI = i
			break LOOP
		case i == 0 && (curY-1) < int(s.lineNo):
			curI = 0
			break LOOP
		case i > 0 && (curY-1) < int(s.lineNo):
			curI = i - 1
			break LOOP
		case i == len(lasses)-1 && (curY-1) > int(s.lineNo):
			curI = i
			break LOOP
		}
	}

	// convert the sasses into an array of characters
	type charPos struct {
		hasPT bool
		pt    playbackTime
	}
	curPosInCharPoss := 0
	charPoss := []charPos{}
	for i, s := range lasses {
		maxChars := int((s.sas.totalHumps() * charsToaHump) + 0.00001) // float rounding
		for j := 0; j < maxChars; j++ {
			cp := charPos{}
			if s.sas.hasPlaybackTime && s.sas.ptCharPosition == j {
				cp.hasPT = true
				cp.pt = s.sas.pt
			}
			charPoss = append(charPoss, cp)

			if i == curI && j == curX-1 {
				curPosInCharPoss = len(charPoss) - 1
			}
		}
	}

	// shortcut if on a playback time
	if charPoss[curPosInCharPoss].hasPT {
		cp := charPoss[curPosInCharPoss]
		fmt.Printf(cp.pt.str)
		return nil
	}

	// get the first and last playback times surrounding the cursor
	var ptFirst, ptLast playbackTime
	ptFirstFound, ptLastFound := false, false
	charsBetweenCurAndFirstPT, charsBetweenCurAndLastPT := 0, 0
	for i := curPosInCharPoss; i >= 0; i-- {
		cp := charPoss[i]
		if cp.hasPT {
			ptFirst = cp.pt
			ptFirstFound = true
			break
		}
		charsBetweenCurAndFirstPT++
	}
	for i := curPosInCharPoss; i < len(charPoss); i++ {
		cp := charPoss[i]
		if cp.hasPT {
			ptLast = cp.pt
			ptLastFound = true
			break
		}
		charsBetweenCurAndLastPT++
	}
	if !ptFirstFound || !ptLastFound {
		fmt.Printf("BAD-PLAYBACK-TIME")
		return nil
	}
	totalCharsBetweenFirstAndLastPT := charsBetweenCurAndFirstPT + charsBetweenCurAndLastPT

	// determine the duration of time passing per hump
	totalDur := ptLast.t.Sub(ptFirst.t)
	durPerChar := float64(totalDur) / float64(totalCharsBetweenFirstAndLastPT)

	// determine the playback time at the current hump
	elapsedFromPtFirst := time.Duration(float64(charsBetweenCurAndFirstPT) * durPerChar)
	ptOut := ptFirst.AddDur(elapsedFromPtFirst)
	fmt.Printf(ptOut.str)
	return nil

	/* OLD CODE for cursor movement (abandoned)
	// potentially have to move more than one
	// character if this calculation has taken to long
	charMovements := 1
	var finalSleep time.Duration
	if diffTime <= durPerOneForthHump {
		finalSleep = durPerOneForthHump - diffTime
	} else {
		charMovements += int(diffTime / durPerOneForthHump)
		finalSleep = diffTime % durPerOneForthHump
	}
	time.Sleep(finalSleep)

	finalCurX, finalCurY, endReached := lasses.getNextPosition(
		curX, middleSasIndex, charMovements)

	// set the position of the cursor
	// -1 because so that first position is 1
	if endReached {
		output = fmt.Sprintf("END")
		return output, nil
	}

	output = fmt.Sprintf("%vgg%vlR \\<esc>", finalCurY, (finalCurX - 1))
	return output, nil
	*/
}

var (
	// line offset from the top of a text-sine (with chords and
	// everything) to the middle of the text-based-sine-curve
	lineNoOffsetToMiddleHump = 3

	charsToaHump = 4.0 // 4 character positions to a hump in a text-based sine wave
)

type lineAndSas struct {
	lineNo int16
	sas    singleAnnotatedSine
}

type lineAndSasses []lineAndSas

// gets the next position of the cursor which is moving hump-movements
// along the text-sine-curve. If there are not enough positions within
// the current hump, then recurively call for the next hump
func (ls lineAndSasses) getNextPosition(startCurX, startHumpIndex, charMovement int) (
	endCurX, endCurY int, endReached bool) {

	if startHumpIndex >= len(ls) {
		return 0, 0, true
	}

	// get the current line hump
	clh := ls[startHumpIndex].sas.totalHumps()

	if startCurX+charMovement <= int(clh*charsToaHump) {
		endCurX = startCurX + charMovement
		endCurY = int(ls[startHumpIndex].lineNo) + lineNoOffsetToMiddleHump
		return endCurX, endCurY, false
	}

	reducedCM := (startCurX + charMovement - int(clh*charsToaHump))
	return ls.getNextPosition(0, startHumpIndex+1, reducedCM)
}

func songsheetFilledCmd(cmd *cobra.Command, args []string) error {

	pdf := gofpdf.New("P", "in", "Letter", "")
	pdf.SetMargins(0, 0, 0)
	pdf.AddPage()

	// each line of text from the input file
	// is attempted to be fit into elements
	// in the order provided within elemKinds
	elemKinds := []tssElement{
		singleSpacing{},
		chordChart{},
		singleAnnotatedSine{},
		singleLineMelody{},
		singleLineLyrics{},
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

// ---------------------

type Pdf interface {
	SetLineWidth(width float64)
	SetLineCapStyle(styleStr string)
	Polygon(points []gofpdf.PointType, styleStr string)
	Line(x1, y1, x2, y2 float64)
	SetFont(familyStr, styleStr string, size float64)
	Text(x, y float64, txtStr string)
	Circle(x, y, r float64, styleStr string)
	Curve(x0, y0, cx, cy, x1, y1 float64, styleStr string)
}

// dummyPdf fulfills the interface Pdf (DNETL)
type dummyPdf struct{}

var _ Pdf = dummyPdf{}

func (d dummyPdf) SetLineWidth(width float64)                            {}
func (d dummyPdf) SetLineCapStyle(styleStr string)                       {}
func (d dummyPdf) Polygon(points []gofpdf.PointType, styleStr string)    {}
func (d dummyPdf) Line(x1, y1, x2, y2 float64)                           {}
func (d dummyPdf) SetFont(familyStr, styleStr string, size float64)      {}
func (d dummyPdf) Text(x, y float64, txtStr string)                      {}
func (d dummyPdf) Circle(x, y, r float64, styleStr string)               {}
func (d dummyPdf) Curve(x0, y0, cx, cy, x1, y1 float64, styleStr string) {}

// ---------------------

// whole text songsheet element
type tssElement interface {
	printPDF(Pdf, bounds) (reduced bounds)
	parseText(lines []string) (reducedLines []string, elem tssElement, err error)
}

// ---------------------

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
func determineChordsSubscriptSuperscript(ch1, ch2, ch3 rune) (subscript, superscript rune) {
	if !(unicode.IsLetter(ch1) && unicode.IsUpper(ch1)) {
		return ' ', ' '
	}
	subscript, superscript = ' ', ' '
	if unicode.IsNumber(ch2) || (unicode.IsLetter(ch2) && unicode.IsLower(ch2)) {
		subscript = ch2
	}
	if unicode.IsNumber(ch3) || (unicode.IsLetter(ch3) && unicode.IsLower(ch3)) {
		superscript = ch3
	}
	return subscript, superscript
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
			yMod := y + spacing/4 + melodyHPadding/2
			yModMid := yMod + melodyHPadding*2
			if i == 0 { // above
				yMod = y - spacing/4 - melodyHPadding/2
				yModMid = yMod - melodyHPadding*2
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

		subscriptCh, superscriptCh := determineChordsSubscriptSuperscript(
			ch1, ch2, ch3)

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

// ---------------------

type singleSpacing struct{}

var _ tssElement = singleSpacing{}

func (s singleSpacing) parseText(lines []string) (reduced []string, elem tssElement, err error) {
	if len(lines) < 1 {
		return lines, elem,
			fmt.Errorf("improper number of input lines, want 1 have %v", len(lines))
	}
	if len(strings.TrimSpace(lines[0])) != 0 {
		return lines, elem, errors.New("blank line contains content")
	}
	return lines[1:], singleSpacing{}, nil
}

func (s singleSpacing) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {
	lineHeight := GetFontHeight(lyricFontPt) * spacingRatioFlag
	return bounds{bnd.top + lineHeight, bnd.left, bnd.bottom, bnd.right}
}

// ---------------------
type singleLineMelody struct {
	melodies []melody
}

var _ tssElement = singleLineMelody{}

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

func (s singleLineMelody) parseText(lines []string) (reduced []string, elem tssElement, err error) {
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

	slm := singleLineMelody{}
	melodiesFound := false
	for i, r := range melodyNums {
		if !(unicode.IsSpace(r) || unicode.IsNumber(r)) {
			return lines, elem, fmt.Errorf(
				"melodies line contains something other"+
					"than numbers and spaces (rune: %v, col: %v)", r, i)
		}
		if unicode.IsSpace(r) {
			slm.melodies = append(slm.melodies, melody{blank: true})
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

		slm.melodies = append(slm.melodies, m)
		melodiesFound = true
	}

	if !melodiesFound {
		return lines, elem, fmt.Errorf("no melodies found")
	}

	return lines[3:], slm, nil
}

func (s singleLineMelody) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {

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
	for i, melody := range s.melodies {
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
			for ; j < len(s.melodies); j++ {
				nmn := s.melodies[j].num
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
			for ; j < len(s.melodies); j++ {
				nmn := s.melodies[j].num
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

// ---------------------

type singleLineLyrics struct {
	lyrics string
}

var _ tssElement = singleLineLyrics{}

func (s singleLineLyrics) parseText(lines []string) (reduced []string, elem tssElement, err error) {
	if len(lines) < 1 {
		return lines, elem,
			fmt.Errorf("improper number of input lines,"+
				" want 1 have %v", len(lines))
	}

	sll := singleLineLyrics{}
	sll.lyrics = lines[0]
	return lines[1:], sll, nil
}

func (s singleLineLyrics) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {

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

// ---------------------

const ( // empirically determined
	ptToHeight    = 100  //72
	widthToHeight = 0.82 //
)

func GetFontPt(heightInches float64) float64 {
	return heightInches * ptToHeight
}

func GetFontHeight(fontPt float64) (heightInches float64) {
	return fontPt / ptToHeight
}

func GetCourierFontWidthFromHeight(height float64) float64 {
	return widthToHeight * height
}

func GetCourierFontHeightFromWidth(width float64) float64 {
	return width / widthToHeight
}

// ---------------------

// TODO
// - slides between chords (central text):
//   - use '\' symbol to indicate slide
//   - faded grey text between this chord and next
//   - can use pdf.SetAlpha()
// - new annotation for hammeron
//   - don't need the whole chord, just the string and number
//     and maybe an 'h' letter in the along sine annotation
//   - ~3 or -3 would mean 'above the number' vs 3~ 3- meaning 'below'
//   - still good to keep the existing 'whole chord hammeron'
//     for any kind of hammeron containing more than one note
//   - Maybe redesign the "central text line" to be the same
//       as the melody line potentially using 2 or 3 lines
//       - OR maybe not, just need ~3 to be inserted at the ~ character... should be okay!
// - move on-sine annotations to either the top or bottom if they
//   directly intersect with a central-axis annotation
// - create an amplitude decay factor (flag) allow for decays
//   to happen in the middle of sine
//    - also allow for pauses (no sine at all)
// - use of sine instead of cos with different text hump pattern:   _
//                                                                 / \_

type singleAnnotatedSine struct {
	hasPlaybackTime bool
	pt              playbackTime
	ptCharPosition  int // number of humps to the playback position
	humps           float64
	trailingHumps   float64 // the sine curve reduces its amplitude to zero during these
	alongAxis       []sineAnnotation
	alongSine       []sineAnnotation
}

func (sas singleAnnotatedSine) totalHumps() float64 {
	return sas.humps + sas.trailingHumps
}

type sineAnnotation struct {
	position    float64 // in humps
	bolded      bool    // whether the whole unit is bolded
	ch          rune    // main character
	subscript   rune    // following subscript character
	superscript rune    // following superscript character
}

// NewsineAnnotation creates a new sineAnnotation object
func NewSineAnnotation(position float64, bolded bool,
	ch, subscript, superscript rune) sineAnnotation {
	return sineAnnotation{
		position:    position,
		bolded:      bolded,
		ch:          ch,
		subscript:   subscript,
		superscript: superscript,
	}
}

var _ tssElement = singleAnnotatedSine{}

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

func GetSASFromTopLines(lines []string) (sas singleAnnotatedSine, err error) {

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
		sas = singleAnnotatedSine{
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

func (s singleAnnotatedSine) parseText(lines []string) (reduced []string, elem tssElement, err error) {

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

		if unicode.IsLetter(ch) &&
			unicode.IsUpper(ch) {

			bolded = true
		}

		ch2, ch3 := ' ', ' '
		if pos+1 < len(fl) {
			ch2 = rune(fl[pos+1])
		}
		if pos+2 < len(fl) {
			ch3 = rune(fl[pos+2])
		}

		subscript, superscript := determineChordsSubscriptSuperscript(
			ch, ch2, ch3)

		alongAxis = append(alongAxis,
			NewSineAnnotation(float64(pos)/4, bolded, ch,
				subscript, superscript))

		if subscript != ' ' {
			pos++
		}
		if superscript != ' ' {
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
			NewSineAnnotation(float64(pos)/4, bolded, ch, ' ', ' '))
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

func (s singleAnnotatedSine) printPDF(pdf Pdf, bnd bounds) (reduced bounds) {

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
	fontPt := GetFontPt(fontH)
	fontHSubSup := fontH * subsupSizeMul
	fontPtSubSup := GetFontPt(fontHSubSup)

	XsubsupCrunch := fontW * 0.1 // squeeze the sub and super script into the chord a bit

	for _, aa := range s.alongAxis {

		X := xStart + (aa.position/s.humps)*width - fontW/2
		Y := yStart + fontH/2 // so the text is centered along the sine axis
		bolded := ""
		if aa.bolded {
			bolded = "B"
		}
		pdf.SetFont("courier", bolded, fontPt)
		pdf.Text(X, Y, string(aa.ch))

		// print sub or super script if exists
		if aa.subscript != ' ' || aa.superscript != ' ' {
			Xsubsup := X + fontW - XsubsupCrunch
			pdf.SetFont("courier", bolded, fontPtSubSup)
			if aa.subscript != ' ' {
				Ysub := Y - fontH/2 + fontHSubSup
				pdf.Text(Xsubsup, Ysub, string(aa.subscript))
			}
			if aa.superscript != ' ' {
				Ysuper := Y - fontH/2
				pdf.Text(Xsubsup, Ysuper, string(aa.superscript))
			}
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
			pdf.Text(tipX-(w/2), tipY+(h/2), string(as.ch))
		}
	}

	return bounds{bnd.top + usedHeight, bnd.left, bnd.bottom, bnd.right}
}
