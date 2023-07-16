package main

import (
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var configPath = flag.String("config", "config.toml", "Path to config file")

func main() {
	flag.Parse()

	// zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load the mapping of JSON-RPC methods to backend URLs
	conf := LoadConfig(*configPath)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Create a new proxy with the providers and the method-to-URL map
	p := NewProxy(conf)

	// Start the proxy
	p.Start()
}
