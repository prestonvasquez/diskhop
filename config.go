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

package diskhop

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Config represents the configuration for the diskhop application.
type Config struct {
	ConnString    string   `yaml:"connString"`              // Remote host
	KeyFile       string   `yaml:"keyFile,omitempty"`       // Path to private key
	Branches      []string `yaml:"branches,omitempty"`      // Branches to sync
	CurrentBranch string   `yaml:"currentBranch,omitempty"` // Current branch
	DB            string   `yaml:"db,omitempty"`            // Database

	// Metadata
	CurDir string `yaml:"-"`
}

// IsDiskhopRepository will check to see if the existing directory contains a
// ".diskhop" configuration file. If it does not, then this function will return
// false.
func IsDiskhopRepository(path string) bool {
	configFilePath := filepath.Join(path, ".diskhop")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return false
	}

	return true
}

// LoadConfig will load the configuration file from the current working
// directory.
// Get the current working directory
func LoadConfig(path string) (Config, error) {
	// Read the config file.
	diskhopFilePath := filepath.Join(path, ".diskhop")

	cbytes, err := os.ReadFile(filepath.Clean(diskhopFilePath))
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the config file.
	cfg := Config{CurDir: path}

	err = yaml.Unmarshal(cbytes, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return cfg, nil
}
