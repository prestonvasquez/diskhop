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

	"github.com/spf13/cobra"
)

func newBranchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "perform branch operations",
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runBranch(cmd, args); err != nil {
			fmt.Println("failed to branch:", err)
		}
	}

	return cmd
}

func runBranch(_ *cobra.Command, args []string) error {
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

	// List all branches, indent once and put a "*" next to the current branch.
	// Highlight the current branch.
	for _, branch := range cfg.Branches {
		if branch == cfg.CurrentBranch {
			// ANSI escape code for red color
			red := "\033[32m"
			// ANSI escape code to reset the color
			reset := "\033[0m"

			// Print the string in red
			fmt.Println(red+" * ", branch, reset)
		} else {
			fmt.Printf("    %s\n", branch)
		}
	}

	return nil
}
