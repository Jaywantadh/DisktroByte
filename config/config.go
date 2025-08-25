package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// AppConfig holds the application-level configuration
type AppConfig struct {
	NodeID           string `mapstructure:"node_id"`
	Port             int    `mapstructure:"port"`
	StoragePath      string `mapstructure:"storage_path"`
	ParallelismRatio int    `mapstructure:"parallelism_ratio"`
}

var Config *AppConfig


func LoadConfig(path string) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)
	viper.AutomaticEnv()

	viper.SetDefault("node_id", "disktrobyte-default-node")
	viper.SetDefault("port", 8080)
	viper.SetDefault("storage_path", "./data")
	viper.SetDefault("parallelism_ratio", 2)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("⚠️ Could not read config file, using defaults: %v", err)
	}

	var appConfig AppConfig
	if err := viper.Unmarshal(&appConfig); err != nil {
		log.Fatalf("❌ Unable to decode config into struct: %v", err)
	}

	Config = &appConfig

	fmt.Println("✅ Configuration loaded successfully.")
}
