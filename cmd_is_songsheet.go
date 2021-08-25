package main

import (
	"fmt"
	"strings"

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
	IsSongsheetCmd = &cobra.Command{
		Use:   "is-ss [filepath]",
		Short: "print TRUE or FALSE if the file is a songsheet",
		Args:  cobra.ExactArgs(1),
		RunE:  isSongsheetCmd,
	}
)

func init() {
	RootCmd.AddCommand(IsSongsheetCmd)
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
