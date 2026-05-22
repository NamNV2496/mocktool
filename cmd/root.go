package cmd

import "github.com/spf13/cobra"

func Execute() error {
	rootCmd := &cobra.Command{
		Short: "A simple mock tool",
	}
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(mcpServerCmd)
	rootCmd.AddCommand(slackBotCmd)
	return rootCmd.Execute()
}
