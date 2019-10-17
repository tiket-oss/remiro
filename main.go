package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/tiket-libre/remiro/handler"
)

func setLogLevel(logLevel string) {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		level = log.WarnLevel
	}
	log.SetLevel(level)
}

func main() {
	var host, port, instruPort, configPath, logLevel string

	flag.StringVarP(&host, "host", "h", "127.0.0.1", "server host address")
	flag.StringVarP(&port, "port", "p", "6379", "port the server will listen to")
	flag.StringVarP(&instruPort, "instru-port", "i", "8888", "configure the port for providing instrumentation")
	flag.StringVarP(&configPath, "config", "c", "config.toml", "configuration file to use")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "Set the logging level (trace, debug, info, warn, error, fatal, panic)")

	flag.Parse()

	setLogLevel(logLevel)
	config, _ := readConfig(configPath)
	addr := fmt.Sprintf("%s:%s", host, port)
	redisHandler := handler.NewRedisHandler(config)
	instruAddr := fmt.Sprintf("%s:%s", host, instruPort)

	instruErr := make(chan error)
	if err := handler.RunInstrumentation(instruAddr, redisHandler, instruErr); err != nil {
		log.Fatalf("Failed to run instrumentation server: %v", err)
	}

	if err := handler.Run(addr, redisHandler); err != nil {
		log.Fatal(err)
	}

	log.Warn(<-instruErr)
}

func readConfig(configPath string) (handler.RedisConfig, error) {
	var config handler.RedisConfig
	_, err := toml.DecodeFile(configPath, &config)

	return config, err
}
