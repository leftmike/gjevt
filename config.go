package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Config struct {
	APIKey  string `hcl:"api_key"`
	BaseURL string `hcl:"base_url,optional"`
	Model   string `hcl:"model,optional"`
}

const defaultConfig = `api_key  = ""
base_url = "https://openrouter.ai/api"
model    = "~typesafe/jev-latest"
`

func loadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(defaultConfig), 0600); err != nil {
			return Config{}, err
		}
		return Config{}, fmt.Errorf("created %s: add your OpenRouter api_key", path)
	} else if err != nil {
		return Config{}, err
	}
	return parseConfig(path, b)
}

func parseConfig(path string, b []byte) (Config, error) {
	var cfg Config
	if err := hclsimple.Decode(path, b, nil, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("%s: api_key is not set", path)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://openrouter.ai/api"
	}
	if cfg.Model == "" {
		cfg.Model = "~typesafe/jev-latest"
	}
	return cfg, nil
}
