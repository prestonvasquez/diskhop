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

	"github.com/prestonvasquez/diskhop"
	"github.com/spf13/cobra"
)

func newCleanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "zero files from bucket",
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runClean(cmd, args); err != nil {
			log.Fatalf("failed to clean: %v", err)
		}
	}

	return cmd
}

func runClean(cmd *cobra.Command, args []string) error {
	curDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Do nothing if we are not in a diskhop repository.
	if !isDiskhopRepository(curDir) {
		return errNotDiskhop
	}

	// Get the files in the directory.
	f, err := os.Open(curDir)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}

	defer f.Close()

	// Read the directory contents
	entities, err := f.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read directory contents: %w", err)
	}

	if len(entities) == 0 {
		return nil
	}

	if err := diskhop.Clean(entities); err != nil {
		return fmt.Errorf("failed to clean: %w", err)
	}

	return nil
}
