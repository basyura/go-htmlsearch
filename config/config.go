package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Port    int    `json:"port"`
	DbFile  string `json:"dbfile"`
	BaseUrl string `json:"baseurl"`
}

func NewConfig() (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(exePath)
	path := filepath.Join(dir, "config.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
