// Package cmd provides CLI support for remiro.
package main

import (
	"github.com/tiket-libre/remiro"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "remiro",
	Short: "Remiro provides service to manipulate request across several redis instances",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := remiro.Config{Port: "9000"}
		handler := remiro.NewRedisHandler().Serve

		remiro.Run(cfg, handler)
	},
}
