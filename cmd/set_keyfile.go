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
	"log"

	"github.com/spf13/cobra"
)

// newSetKeyFileCommand creates a new cobra command for setting the keyfile name
// in the configuration.
func newSetKeyFileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key-file",
		Short: "Set the keyfile name in the configuration",
		Args:  cobra.ExactArgs(1), // Ensures exactly one argument is provided
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runSet(cmd, args, func(cfg *config) error {
			cfg.KeyFile = args[0]

			return nil
		}); err != nil {
			log.Fatalf("failed to set keyfile: %v", err)
		}
	}

	return cmd
}
