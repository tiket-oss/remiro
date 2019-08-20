// Package cmd provides CLI support for remiro.
package cmd

import (
	"log"

	"github.com/tiket-libre/remiro"
	"github.com/tiket-libre/remiro/server"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "remiro",
	Short: "Remiro provides service to manipulate request across several redis instances",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := server.Config{Port: "9000"}
		handler := remiro.NewRedisHandler().Serve

		server.Run(cfg, handler)
	},
}

// Execute is a wrapper that will initialize root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
