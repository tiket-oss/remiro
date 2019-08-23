package main

import (
	"fmt"
	"log"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/tiket-libre/remiro"
)

func main() {
	var host, port, configPath string

	flag.StringVarP(&host, "host", "h", "127.0.0.1", "server host address")
	flag.StringVarP(&port, "port", "p", "6379", "port the server will listen to")
	flag.StringVarP(&configPath, "config", "c", "config.json", "configuration file to use")

	flag.Parse()

	config, _ := readConfig(configPath)

	addr := fmt.Sprintf("%s:%s", host, port)
	handler := remiro.NewRedisHandler(config)
	if err := remiro.Run(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func readConfig(configPath string) (remiro.RedisConfig, error) {
	config := remiro.RedisConfig{
		DeleteOnGet: true,
		DeleteOnSet: true,
		Source: remiro.ClientConfig{
			Addr:         "127.0.0.1:6380",
			MaxIdleConns: 50,
			IdleTimeout:  30 * time.Second,
		},
		Destination: remiro.ClientConfig{
			Addr:         "127.0.0.1:6381",
			MaxIdleConns: 100,
			IdleTimeout:  45 * time.Second,
		},
	}

	return config, nil
}
