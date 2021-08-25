package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

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
	SongsheetPlaybackTimeCmd = &cobra.Command{
		Use:   "pt [filepath] [cursor-x] [cursor-y]",
		Short: "return the playback time (mm:ss.cs) for the current position",
		Args:  cobra.ExactArgs(3),
		RunE:  playbackTimeCmd,
	}
)

func init() {
	RootCmd.AddCommand(SongsheetPlaybackTimeCmd)
}

func playbackTimeCmd(cmd *cobra.Command, args []string) error {

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
