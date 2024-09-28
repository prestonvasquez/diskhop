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

type checkoutFlags struct {
	newBranch string
}

func newCheckoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkout",
		Short: "Checkout a branch",
	}

	checkoutFlags := checkoutFlags{}

	cmd.Flags().StringVarP(&checkoutFlags.newBranch, "branch", "b", "", "create a new branch")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runCheckout(cmd, args, checkoutFlags); err != nil {
			log.Fatalf("failed to checkout: %v", err)
		}
	}

	return cmd
}

func checkoutNewBranch(cfg *config, newName string) error {
	// Check to see if the branch is in the cfg object.
	for _, branch := range cfg.Branches {
		if branch == newName {
			return fmt.Errorf("branch already exists: %s", newName)
		}
	}

	// Update the cfg object.
	cfg.Branches = append(cfg.Branches, newName)

	return nil
}

func checkoutBranch(cfg *config, branchName string) error {
	// Check to see if the branch exists.
	found := false
	for _, branch := range cfg.Branches {
		if branch == branchName {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("branch does not exist: %s", branchName)
	}

	// Update the current branch.
	cfg.CurrentBranch = branchName

	return nil
}

func runCheckout(_ *cobra.Command, args []string, flags checkoutFlags) error {
	curDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Do nothing if we are not in a diskhop repository.
	if !isDiskhopRepository(curDir) {
		return errNotDiskhop
	}

	// Read the .diskhop file.
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// If we are just creating a new branch, then we don't need to do anything
	// particularly special.
	if flags.newBranch != "" {
		if err := checkoutNewBranch(&cfg, flags.newBranch); err != nil {
			return fmt.Errorf("failed to create new branch: %w", err)
		}
	} else {
		if len(args) != 1 {
			return fmt.Errorf("branch name required")
		}

		branch := args[0]
		if err := checkoutBranch(&cfg, branch); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	}

	// Write the new config file.
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := os.WriteFile(filepath.Join(curDir, ".diskhop"), bytes, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
