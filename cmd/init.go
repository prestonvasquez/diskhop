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
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func runInit(_ *cobra.Command, _ []string, cfg config) error {
	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// If the .diskhop file already exists, then we don't need to do anything.
	if isDiskhopRepository(filepath.Join(wd, ".diskhop")) {
		return errNotDiskhop
	}

	// Turn the cfg into the .diskhop yaml file.
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(".diskhop", bytes, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// newInitCommand creates a new cobra command for the init operation.
func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new diskhop project",
	}

	cfg := config{
		Branches:      []string{"main"},
		CurrentBranch: "main",
	}

	cmd.Flags().StringVar(&cfg.ConnString, "conn-string", "", "connection string")
	cmd.Flags().StringVar(&cfg.KeyFile, "key", "", "path to private key for CSE")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runInit(cmd, args, cfg); err != nil {
			log.Fatalf("failed to init: %v", err)
		}
	}

	return cmd
}
