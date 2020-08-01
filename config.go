package main

import (
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
)

// Create a default config file if not found config.toml
func loadConfigIfExists() {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("toml")   // REQUIRED if the config file does not have the extension in the name
	//viper.AddConfigPath("/etc/rsync-os/")   // path to look for the config file in
	//viper.AddConfigPath("$HOME/.rsync-os")  // call multiple times to add many search paths
	viper.AddConfigPath(".") // optionally look for config in the working directory

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config File not found

			createSampleConfig()
			log.Fatalln("Config does not exist, a sample of config was created")
		} else {
			// Found but got errors
			log.Fatalln(err)
		}
	}
}

func createSampleConfig() {
	confSample := []byte(
`title = "configuration of rsync-os"

# [object storage's name] 
[minio]
  endpoint = "127.0.0.1:9000"
  keyAccess = "minioadmin"
  keySecret = "minioadmin"
  [minio.boltdb]
    path = "test.db"
`)

	if ioutil.WriteFile("config.toml", confSample, 0666) != nil {
		log.Fatalln("Can't create a sample of config")
	}

}
