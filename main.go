package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/tiket-libre/remiro/handler"
)

func main() {
	var host, port, instruPort, configPath string
	var verbose bool

	flag.StringVarP(&host, "host", "h", "127.0.0.1", "server host address")
	flag.StringVarP(&port, "port", "p", "6379", "port the server will listen to")
	flag.StringVarP(&instruPort, "instru-port", "i", "8888", "configure the port for providing instrumentation")
	flag.StringVarP(&configPath, "config", "c", "config.toml", "configuration file to use")
	flag.BoolVarP(&verbose, "verbose", "v", false, "Set remiro to be verbose, logging every events that happened")
	flag.Parse()

	config, _ := readConfig(configPath)
	redisHandler := handler.NewRedisHandler(config)
	addr := fmt.Sprintf("%s:%s", host, port)
	instruAddr := fmt.Sprintf("%s:%s", host, instruPort)
	if verbose {
		log.SetLevel(log.TraceLevel)
	}

	fmt.Printf("Preparing instrumentation endpoints...\n")
	runInstruErr := make(chan error)
	if err := handler.RunInstrumentation(instruAddr, redisHandler, runInstruErr); err != nil {
		log.Fatalf("Failed to run instrumentation server: %v", err)
	} else {
		go func() { log.Warn(<-runInstruErr) }()
		fmt.Printf("Instrumentation is available at %s\n", instruAddr)
	}

	fmt.Printf("Remiro is now running at %s\n", addr)
	if err := handler.Run(addr, redisHandler); err != nil {
		log.Fatal(err)
	}
}

func readConfig(configPath string) (handler.RedisConfig, error) {
	var config handler.RedisConfig
	_, err := toml.DecodeFile(configPath, &config)

	return config, err
}
