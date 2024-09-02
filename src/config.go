package src

import (
	"cmp"
	"fmt"
	"github.com/go-ini/ini"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Directories []string `ini:"dirs"`
	Name        string   `ini:"name"`
	Port        int      `ini:"port"`
	Host        string   `ini:"host"`
	Tag         string   `ini:"tag"`
}

func SetConfig() *Config {
	config_dir, err := os.UserConfigDir()
	if err != nil {
		config_dir, _ = expandPath("~/.config/")
	}

	// Set the defaults
	home_dir, _ := os.UserHomeDir()
	var cfg Config = Config{Directories: []string{home_dir}, Name: "Browse!", Port: 8080, Host: "localhost", Tag: "Browse local code locally!"}
	config_file := filepath.Join(config_dir, "browse", "config.ini")
	ini_load, err := ini.Load(config_file)
	if err != nil {
		return &cfg
	}

	var cfg2 Config
	err = ini_load.Section("directories").MapTo(&cfg2)
	if err != nil {
		return &cfg
	}

	err = ini_load.Section("settings").MapTo(&cfg2)
	if err != nil {
		return &cfg
	}

	var cdirs []string
	for _, dir := range cfg2.Directories {
		if strings.HasSuffix(dir, "*") {
			expandedPath, err := expandPath(filepath.Dir(dir))
			if err != nil {
				return &cfg
			}

			dirs, err := os.ReadDir(expandedPath)
			if err != nil {
				return &cfg
			}

			for _, dir_s := range dirs {
				cdirs = append(cdirs, fmt.Sprintf("%s/%s", filepath.Dir(dir), dir_s.Name()))
			}
		} else {
			cdirs = append(cdirs, dir)
		}
	}

	cfg2.Directories = uniqueSortedEntries(cdirs)

	// Expand and resolve directories
	var expandedDirs []string
	for _, dir := range cfg2.Directories {
		expandedPath, err := expandPath(dir)
		if err != nil {
			return &cfg
		}
		resolvedPath, err := resolvePath(expandedPath)
		if err != nil {
			return &cfg
		}
		expandedDirs = append(expandedDirs, resolvedPath)
	}

	cfg2.Directories = expandedDirs
	if len(expandedDirs) <= 0 {
		cfg2.Directories = cfg.Directories
	}
	cfg2.Name = cmp.Or(cfg2.Name, cfg.Name)
	cfg2.Port = cmp.Or(cfg2.Port, cfg.Port)
	cfg2.Host = cmp.Or(cfg2.Host, cfg.Host)
	cfg2.Tag = cmp.Or(cfg2.Tag, cfg.Tag)

	return &cfg2
}
