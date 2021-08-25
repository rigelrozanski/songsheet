package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"strings"

	"github.com/spf13/cobra"
)

var (
	SongsheetFillBPM = &cobra.Command{
		Use: "fill-bpm [filepath]",
		Short: "add a line to the beginning of the " +
			"songsheet (at [filepath]) with the average bpm",
		Args: cobra.ExactArgs(1),
		RunE: fillBPMCmd,
	}
)

func init() {
	RootCmd.AddCommand(SongsheetFillBPM)
}

func fillBPMCmd(cmd *cobra.Command, args []string) error {
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
