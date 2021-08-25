package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/rigelrozanski/thranch/quac"
	"github.com/spf13/cobra"
)

var (
	GetAudioCmd = &cobra.Command{
		Use:   "audio-fp [filepath]",
		Short: "print the audio filepath if exists or allocate a new file if it doesn't",
		Args:  cobra.ExactArgs(1),
		RunE:  audioFPCmd,
	}

	HasAudioCmd = &cobra.Command{
		Use:   "has-audio [filepath]",
		Short: "print TRUE or FALSE if the file has associated audio",
		Args:  cobra.ExactArgs(1),
		RunE:  hasAudioCmd,
	}
)

func init() {
	RootCmd.AddCommand(GetAudioCmd)
	RootCmd.AddCommand(HasAudioCmd)
}

const (
	audioLinePrefix = commentPrefix + " AUDIO-ID="
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

func hasAudioCmd(cmd *cobra.Command, args []string) error {
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

func audioFPCmd(cmd *cobra.Command, args []string) error {
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
