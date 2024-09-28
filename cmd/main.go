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

	"github.com/prestonvasquez/diskhop"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:     "dop",
		Short:   "Diskhop is a cli that transfers data between disk storage to a database",
		Version: diskhop.Version,
	}

	cmd.AddCommand(newBranchCommand())
	cmd.AddCommand(newCheckoutCommand())
	cmd.AddCommand(newCleanCommand())
	cmd.AddCommand(newConfigCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newPullCommand())
	cmd.AddCommand(newPushCommand())
	cmd.AddCommand(newRevertCommand())

	if err := cmd.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
