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

	"github.com/spf13/cobra"
)

func newRevertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revert",
		Short: "Revert to a previous commit",
		Args:  cobra.ExactArgs(1),
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runRevert(cmd, args); err != nil {
			log.Fatalf("failed to revert: %v", err)
		}
	}

	return cmd
}

func runRevert(cmd *cobra.Command, args []string) error {
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

	// Geth the pusher for the remote host.
	diskhopStore, err := newDiskhopStore(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("failed to create diskhop store: %w", err)
	}

	if diskhopStore.reverter == nil {
		return fmt.Errorf("store does not support revert")
	}

	if err := diskhopStore.reverter.Revert(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("failed to revert: %w", err)
	}

	return nil
}
