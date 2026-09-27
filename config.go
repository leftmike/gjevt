package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/leftmike/gjevt/jev"
)

type fileConfig struct {
	APIKey  string `hcl:"api_key"`
	BaseURL string `hcl:"base_url,optional"`
	Model   string `hcl:"model,optional"`
}

const defaultConfig = `api_key  = ""
base_url = "` + jev.DefaultBaseURL + `"
model    = "` + jev.DefaultModel + `"
`

func loadConfig(path string) (jev.Config, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(defaultConfig), 0600); err != nil {
			return jev.Config{}, err
		}
		return jev.Config{}, fmt.Errorf("created %s: add your OpenRouter api_key", path)
	} else if err != nil {
		return jev.Config{}, err
	}
	return parseConfig(path, b)
}

func parseConfig(path string, b []byte) (jev.Config, error) {
	var fc fileConfig
	if err := hclsimple.Decode(path, b, nil, &fc); err != nil {
		return jev.Config{}, err
	}
	if fc.APIKey == "" {
		return jev.Config{}, fmt.Errorf("%s: api_key is not set", path)
	}
	return jev.Config(fc), nil
}
