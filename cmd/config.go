// Copyright 2024 Preston Vasquez
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// config represents the configuration for the diskhop application.
type config struct {
	ConnString    string   `yaml:"connString"`              // Remote host
	KeyFile       string   `yaml:"keyFile,omitempty"`       // Path to private key
	Branches      []string `yaml:"branches,omitempty"`      // Branches to sync
	CurrentBranch string   `yaml:"currentBranch,omitempty"` // Current branch
	DB            string   `yaml:"db,omitempty"`            // Database

	// Metadata
	CurDir string `yaml:"-"`
}

// storeType represents the type of store.
type storeType uint8

const (
	// storeTypeUnknown represents an unknown store type.
	storeTypeUnknown storeType = iota

	// storeTypeMongo represents a MongoDB store.
	storeTypeMongo
)

// getAESKey will read the private key from the file system.
func getAESKey(cfg config) ([]byte, error) {
	if cfg.KeyFile == "" {
		return nil, nil
	}

	aesKey, err := os.ReadFile(cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	return aesKey, nil
}

// getStoreType returns the type of store based on the connection string schema.
func getStoreType(cfg config) storeType {
	uri, err := url.Parse(cfg.ConnString)
	if err != nil {
		return storeTypeUnknown
	}

	var stype storeType

	//nolint:gocritic,revive
	switch uri.Scheme {
	case "mongodb":
		stype = storeTypeMongo
	}

	return stype
}

// isDiskhopRepository will check to see if the existing directory contains a
// ".diskhop" configuration file. If it does not, then this function will return
// false.
func isDiskhopRepository(path string) bool {
	configFilePath := filepath.Join(path, ".diskhop")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return false
	}

	return true
}

// loadConfig will load the configuration file from the current working
// directory.
// Get the current working directory
func loadConfig() (config, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return config{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Read the config file.
	diskhopFilePath := filepath.Join(currentDir, ".diskhop")

	cbytes, err := os.ReadFile(filepath.Clean(diskhopFilePath))
	if err != nil {
		return config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the config file.
	cfg := config{CurDir: currentDir}

	err = yaml.Unmarshal(cbytes, &cfg)
	if err != nil {
		return config{}, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return cfg, nil
}

// newConfigCommand creates a new cobra command for managing configuration.
func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration for diskhop",
	}

	cmd.AddCommand(newSetCommand())

	return cmd
}
