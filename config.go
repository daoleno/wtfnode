package main

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	Providers         []string        `mapstructure:"providers"`
	MethodsMapping    []MethodMapping `mapstructure:"methods_mapping"`
	RequestsPerSecond int             `mapstructure:"requests_per_second"`
	Burst             int             `mapstructure:"burst"`
	RetryLimit        int             `mapstructure:"retry_limit"`
	SendBatchDirectly bool            `mapstructure:"send_batch_directly"`
}

type MethodMapping struct {
	Method    string   `mapstructure:"method"`
	Providers []string `mapstructure:"providers"`
}

// LoadConfig loads the mapping of JSON-RPC methods to backend URLs from a configuration file
func LoadConfig(file string) Config {
	viper.SetConfigFile(file)   // set the custom file path
	viper.SetConfigType("toml") // REQUIRED if the config file does not have the extension in the name

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v", err)
	}

	// pretty print the config
	json, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to marshal config")
	}
	fmt.Println("Loaded config:")
	fmt.Println(string(json))

	return config
}
