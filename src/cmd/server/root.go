package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version represents the Version of the kkpctl binary, should be set via ldflags -X
	Version string

	// Date represents the Date of when the kkpctl binary was build, should be set via ldflags -X
	Date string

	// Commit represents the Commit-hash from which kkpctl binary was build, should be set via ldflags -X
	Commit string

	// BuiltBy represents who build the binary, should be set via ldflags -X
	BuiltBy string

	hostname       string
	port           int
	configFileName string
	tsNetStateDir  string
	capName        string
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&configFileName, "config", "c", "", "Name of the config file")

	rootCmd.PersistentFlags().IntVar(&port, "port", 443, "port to listen on")
	viper.SetDefault("server.port", 443)
	err := viper.BindPFlag("server.port", rootCmd.PersistentFlags().Lookup("port"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	rootCmd.PersistentFlags().StringVarP(&hostname, "server", "s", "tka-server", "Port of the gRPC API of the Server")
	viper.SetDefault("server.hostname", "tka-server")
	err = viper.BindPFlag("server.hostname", rootCmd.PersistentFlags().Lookup("server"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	rootCmd.PersistentFlags().StringVar(&tsNetStateDir, "dir", "", "tsnet state directory; a default one will be created if not provided")
	viper.SetDefault("tailscale.stateDir", "")
	err = viper.BindPFlag("tailscale.stateDir", rootCmd.PersistentFlags().Lookup("dir"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	rootCmd.PersistentFlags().StringVar(&capName, "cap-name", "specht-labs.de/cap/tka", "name of the capability to request from tailscale")
	viper.SetDefault("tailscale.capName", "specht-labs.de/cap/tka")
	err = viper.BindPFlag("tailscale.capName", rootCmd.PersistentFlags().Lookup("cap-name"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}
}

func initConfig() {
	if configFileName != "" {
		viper.SetConfigFile(configFileName)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.AddConfigPath("$HOME/.config/tailscale-k8s-auth/")
		viper.AddConfigPath("/data")
	}

	viper.SetEnvPrefix("TKA")
	viper.AutomaticEnv()

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		// Handle errors reading the config file
		//otelzap.L().WithError(err).Warn("Failed to read config file. This might be still valid if you provided all the environment variables or command line flags.")
	}

	hostname = viper.GetString("server.hostname")
	port = viper.GetInt("server.port")
	tsNetStateDir = viper.GetString("tailscale.stateDir")
	capName = viper.GetString("tailscale.capName")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "tka",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
