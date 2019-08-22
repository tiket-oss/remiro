package main

import (
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/tiket-libre/remiro"
)

var rootCmd = &cobra.Command{
	Use:   "remiro",
	Short: "Remiro provides service to manipulate request across several redis instances",
	Run: func(cmd *cobra.Command, args []string) {
		config := remiro.RedisConfig{
			DeleteOnGet: true,
			DeleteOnSet: true,
			Source: remiro.ClientConfig{
				Addr:         "127.0.0.1:6379",
				MaxIdleConns: 50,
				IdleTimeout:  30 * time.Second,
			},
			Destination: remiro.ClientConfig{
				Addr:         "127.0.0.1:6380",
				MaxIdleConns: 100,
				IdleTimeout:  45 * time.Second,
			},
		}

		handler := remiro.NewRedisHandler(config)
		remiro.Run("127.0.0.1:9000", handler)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
