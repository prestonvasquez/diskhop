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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// newSetCommand creates a new cobra command for setting configuration values.
func newSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a configuration value",
	}

	cmd.AddCommand(newSetKeyFileCommand())
	cmd.AddCommand(newSetConnStringCommand())

	return cmd
}

func runSet(_ *cobra.Command, _ []string, set func(*config) error) error {
	// Make sure we are in a diskhop repository
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if !isDiskhopRepository(filepath.Join(wd, ".diskhop")) {
		return errNotDiskhop
	}
	// Load the configuration

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := set(&cfg); err != nil {
		return fmt.Errorf("failed to set configuration: %w", err)
	}

	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(filepath.Join(wd, ".diskhop"), bytes, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
