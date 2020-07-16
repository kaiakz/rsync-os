package main

import (
	"bytes"
	"github.com/spf13/viper"
)

type Config struct {
	/* boltdb */
	// path_boltdb

	/* minio */
	//endpoint := "127.0.0.1:9000"
	//accessKeyID := "minioadmin"
	//secretAccessKey := "minioadmin"
}

func LoadYAMLConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	//viper.AddConfigPath("/etc/rsync-os/")   // path to look for the config file in
	//viper.AddConfigPath("$HOME/.rsync-os")  // call multiple times to add many search paths
	viper.AddConfigPath(".")               // optionally look for config in the working directory

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil { // Handle errors reading the config file
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config File not found
			// Create a default config file
			CreateSampleConfig()
			panic("Config does not exist")
		} else {
			// Found but got errors
		}
	}
}

func CreateSampleConfig() {
	confExample := []byte(`
boltdb:
  path: test.db
minio:
  endpoint: 127.0.0.1:9000
  accessKeyID: minioadmin
  secretAccessKey: minioadmin
`)
	viper.ReadConfig(bytes.NewBuffer(confExample))
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml") // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")    // optionally look for config in the working directory
	viper.SafeWriteConfig()
}