package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/tiket-libre/remiro"
)

func main() {
	var host, port, configPath string

	flag.StringVarP(&host, "host", "h", "127.0.0.1", "server host address")
	flag.StringVarP(&port, "port", "p", "6379", "port the server will listen to")
	flag.StringVarP(&configPath, "config", "c", "config.toml", "configuration file to use")

	flag.Parse()

	config, _ := readConfig(configPath)

	addr := fmt.Sprintf("%s:%s", host, port)
	handler := remiro.NewRedisHandler(config)
	if err := remiro.Run(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func readConfig(configPath string) (remiro.RedisConfig, error) {
	var config remiro.RedisConfig
	_, err := toml.DecodeFile(configPath, &config)

	return config, err
}
