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

	"crypto/aes"
	"crypto/cipher"

	"github.com/prestonvasquez/diskhop"
	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"github.com/prestonvasquez/diskhop/store"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

const defaultSampeSize = 5

func runPull(cmd *cobra.Command, _ []string, opts store.PullOptions) error {
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

	// Get the files in the directory.
	f, err := os.Open(curDir)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}

	defer f.Close()

	// Read the directory contents
	fileInfo, _ := f.Readdir(-1)

	if err := diskhop.Clean(fileInfo); err != nil {
		return fmt.Errorf("failed to clean directory: %w", err)
	}

	// Get the AEAD key, if it exists.
	key, err := getAESKey(cfg)
	if err != nil {
		return fmt.Errorf("failed to get AES key from config: %w", err)
	}

	defer dcrypto.Zero(key)

	// Geth the pusher for the remote host.
	diskhopStore, err := newDiskhopStore(cmd.Context(), cfg)
	if err != nil {
		return fmt.Errorf("failed to create diskhop store: %w", err)
	}

	dp := diskhop.NewFilePuller(diskhopStore.puller)

	trackerDone := make(chan struct{}, 1)
	go func() {
		defer close(trackerDone)

		total := <-dp.Total()
		bar := progressbar.NewOptions(total,
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionSetDescription("[cyan][1/1][reset] Pulling data..."),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))

		for range dp.Progress() {
			bar.Add(1)
		}
	}()

	pullOpts := []store.PullOption{
		func(o *store.PullOptions) {
			*o = opts
		},
	}

	if key != nil {
		block, err := aes.NewCipher(key)
		if err != nil {
			return fmt.Errorf("failed to create new AES cipher: %w", err)
		}

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			return fmt.Errorf("failed to create new GCM cipher: %w", err)
		}

		so := dcrypto.NewAEAD(diskhopStore.ivMgr, aesgcm)

		pullOpts = append(pullOpts, store.WithPullSealOpener(so))
	}

	if err := dp.Pull(cmd.Context(), pullOpts...); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	<-trackerDone

	return nil
}

// newPullCommand creates a new cobra command for the pull subcommand to pull
// files from the remote host.
func newPullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "pull",
		// Args: cobra.ExactArgs(1), // Ensures exactly one argument is provided
		Long: "pull will download files from the remote host to a local diskhop directory",
	}

	flags := store.PullOptions{}

	cmd.Flags().IntVar(&flags.SampleSize, "sample", defaultSampeSize, "chose a random subset of data")
	cmd.Flags().StringVarP(&flags.Filter, "filter", "f", "", "filter documents by expression")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runPull(cmd, args, flags); err != nil {
			log.Fatalf("failed to pull: %v", err)
		}
	}

	return cmd
}
