package config

import (
	"os"
	"path"
	"encoding/json"
//	"fmt"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	Db_url string `json:"db_url"`
	Current_user_name string `json:"current_user_name"`
}

func (cfg *Config) SetUser(username string) error {
	cfg.Current_user_name = username
	return write(*cfg)
}

func Read() (Config, error) {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	jsonData, err := os.ReadFile(configFilePath)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	err = json.Unmarshal(jsonData, &cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func write(cfg Config) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	fi, err := os.Stat(configFilePath)
	if err != nil {
		return err
	}

	cfgData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	
	err = os.WriteFile(configFilePath, cfgData, fi.Mode().Perm())
	if err != nil {
		return err
	}

	return nil
}

func getConfigFilePath() (string, error) {
	userHomePath, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return path.Join(userHomePath, configFileName), nil
}
