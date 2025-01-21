package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	TestEndpoint string `json:"testEndpoint"`
}

var appConfig = Config{
	TestEndpoint: "http://0.0.0.0:5000/apptest", // default value
}

func LoadConfig() error {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if (err != nil) {
		if os.IsNotExist(err) {
			return SaveConfig() // Save default config if file doesn't exist
		}
		return err
	}

	return json.Unmarshal(data, &appConfig)
}

func SaveConfig() error {
	configPath := getConfigPath()
	data, err := json.MarshalIndent(appConfig, "", "    ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func getConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "inventoryt-printer", "config.json")
}

func SetTestEndpoint(url string) error {
	appConfig.TestEndpoint = url
	return SaveConfig()
}
