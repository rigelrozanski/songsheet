package main

import (
	"fmt"
	"os"

	"github.com/rigelrozanski/thranch/quac"
	"github.com/spf13/cobra"
)

func main() {
	quac.Initialize(os.ExpandEnv("$HOME/.thranch_config"))
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "mt",
	Short: "multitool, a collection of handy lil tools",
}
