package main

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tiket-libre/remiro"
)

var rootCmd = &cobra.Command{
	Use:   "remiro",
	Short: "Remiro provides service to manipulate request across several redis instances",
	Run: func(cmd *cobra.Command, args []string) {
		handler := remiro.NewRedisHandler("127.0.0.1:6379", "127.0.0.1:6380")
		remiro.Run("127.0.0.1:9000", handler)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
