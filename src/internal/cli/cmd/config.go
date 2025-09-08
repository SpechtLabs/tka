package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func initConfig() {
	if configFileName != "" {
		viper.SetConfigFile(configFileName)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("$HOME/.config/tka/")
		viper.AddConfigPath("/etc/tka/")
	}

	viper.SetEnvPrefix("TKA")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	// Find and read the config file (optional). If not found, continue; if malformed, print and exit.
	if err := viper.ReadInConfig(); err != nil {
		if _, notFound := err.(viper.ConfigFileNotFoundError); notFound {
			return
		}
		fmt.Fprintf(os.Stderr, "error reading config file: %v\n", err)
		os.Exit(2)
	}
}
