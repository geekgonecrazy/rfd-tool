package main

import (
	"flag"
	"log"

	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/core"
	"github.com/geekgonecrazy/rfd-tool/router"
)

func main() {
	configFile := flag.String("configFile", "config.yaml", "Config File full path. Defaults to current folder")

	flag.Parse()

	if err := config.Load(*configFile); err != nil {
		log.Fatalln("Error loading Config file: ", err)
	}

	if err := core.Setup(); err != nil {
		log.Fatalln("Failed initializing core: ", err)
	}

	if err := router.Run(); err != nil {
		log.Fatalln("Failed to start router: ", err)
	}

}
