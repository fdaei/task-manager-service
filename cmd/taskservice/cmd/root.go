package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "taskservice",
	Short: "Taskservice CLI entrypoint",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize()
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}
