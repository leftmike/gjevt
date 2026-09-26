package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want Config
		err  string
	}{
		{
			name: "defaults",
			src:  `api_key = "k"`,
			want: Config{APIKey: "k", BaseURL: "https://openrouter.ai/api", Model: "~typesafe/jev-latest"},
		},
		{
			name: "all fields",
			src:  "api_key = \"k\"\nbase_url = \"http://x\"\nmodel = \"typesafe/jev-1.13\"\n",
			want: Config{APIKey: "k", BaseURL: "http://x", Model: "typesafe/jev-1.13"},
		},
		{name: "empty key", src: defaultConfig, err: "api_key is not set"},
		{name: "missing key", src: `model = "m"`, err: "api_key"},
		{name: "unknown field", src: "api_key = \"k\"\nbogus = 1\n", err: "bogus"},
		{name: "syntax", src: `api_key = `, err: "test.hcl"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseConfig("test.hcl", []byte(c.src))
			if c.err != "" {
				if err == nil || !strings.Contains(err.Error(), c.err) {
					t.Fatalf("got error %v, want %q", err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestLoadConfigCreatesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gjevt.hcl")

	_, err := loadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "created") {
		t.Fatalf("got error %v, want created", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("got mode %v, want 0600", fi.Mode().Perm())
	}

	_, err = loadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "api_key is not set") {
		t.Fatalf("got error %v, want api_key is not set", err)
	}
}
