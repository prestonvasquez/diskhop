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
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/prestonvasquez/diskhop"
	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"github.com/prestonvasquez/diskhop/store"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Check if the argument is "origin"
func validateArg(arg string) error {
	if arg == "origin" {
		return nil
	}

	// Check if the argument matches the pattern "upstream/{name}"
	match, _ := regexp.MatchString(`^migrate/[a-zA-Z0-9-]+$`, arg)
	if match {
		return nil
	}

	// If neither condition is met, return an error
	return fmt.Errorf("invalid argument: %s. Must be 'origin' or 'upstream/{name}'", arg)
}

func extractName(arg string) (string, error) {
	// Regular expression with a capturing group to capture the name part from "migrate/{name}"
	re := regexp.MustCompile(`^migrate/([a-zA-Z0-9-]+)$`)
	matches := re.FindStringSubmatch(arg)
	if len(matches) == 2 {
		return matches[1], nil
	}

	return "", fmt.Errorf("invalid format: %s. Must be 'migrate/{name}'", arg)
}

// pushWithProgress updates one file at a time, overwriting the same log line
func pushWithProgress(dir string, progressCh <-chan store.NameProgress) error {
	// grab Logrus text formatter
	formatter, ok := logrus.StandardLogger().Formatter.(*logrus.TextFormatter)
	if !ok {
		// fallback: simple logging per update
		for pr := range progressCh {
			logrus.Infof("%s: %6.2f%%", pr.Name, pr.Progress)
		}
		return nil
	}

	// Get the number of files in the dir.
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Don't count hidden files.
	var fileCount int
	for _, file := range files {
		if file.Name()[0] != '.' {
			fileCount++
		}
	}

	logrus.Infof("📤 Pushing data\n")

	var oldName string
	count := 1

	type nameProgressKey struct {
		Name     string
		Progress float64
	}

	// consume progress events
	for pr := range progressCh {
		if oldName != "" && oldName != pr.Name {
			// Break for each new file.
			os.Stdout.Write([]byte("\n"))
			count++
		}

		oldName = pr.Name

		// build a formatted log entry
		entry := &logrus.Entry{
			Logger:  logrus.StandardLogger(),
			Data:    logrus.Fields{},
			Time:    time.Now(),
			Level:   logrus.InfoLevel,
			Message: fmt.Sprintf("  [%d/%d] %s: %6.2f%%", count, fileCount, pr.Name, pr.Progress),
		}
		lineBytes, err := formatter.Format(entry)
		if err != nil {
			continue
		}
		line := strings.TrimRight(string(lineBytes), "\n")

		// Carriage return + overwrite
		os.Stdout.Write([]byte("\r" + line))
	}

	// after done, move to next line
	os.Stdout.Write([]byte("\n"))
	return nil
}

func runPush(cmd *cobra.Command, args []string, opts store.PushOptions) error {
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

	// Get the AEAD key, if it exists.
	key, err := getAESKey(cfg)
	if err != nil {
		return fmt.Errorf("failed to get AES key from config: %w", err)
	}

	defer dcrypto.Zero(key)

	var diskhopStore *diskhopStore
	if args[0] == "origin" {
		// Geth the pusher for the remote host.
		diskhopStore, err = newDiskhopStore(cmd.Context(), cfg)
		if err != nil {
			return fmt.Errorf("failed to create diskhop store: %w", err)
		}
	} else {
		diskhopStore, err = newDiskhopStoreUpstream(cmd.Context(), args[0], cfg)
		if err != nil {
			return fmt.Errorf("failed to create diskhop store: %w", err)
		}
	}

	dopPusher := diskhop.NewFilePusher(diskhopStore.pusher)

	// Get the files in the directory.
	f, err := os.Open(curDir)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}

	defer f.Close()

	pushOpts := []store.PushOption{
		func(o *store.PushOptions) {
			*o = opts
		},
	}

	progressCh := make(chan store.NameProgress)
	pushOpts = append(pushOpts, store.WithPushNameProgress(progressCh))

	go pushWithProgress(curDir, progressCh)

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

		pushOpts = append(pushOpts, store.WithPushSealOpener(so))
	}

	if err := dopPusher.Push(cmd.Context(), f, pushOpts...); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// newPushCommand creates a new cobra command for the push operation.
func newPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "push",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("this command requires exactly one argument")
			}

			return validateArg(args[0])
		},
		Long: "upsert the files from the local diskhop directory to remote host",
	}

	flags := store.PushOptions{}

	cmd.Flags().IntVar(&flags.RetryPolicy.MaxRetries, "retries", 3, "number of retries to attempt on transient errors")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		if err := runPush(cmd, args, flags); err != nil {
			log.Fatalf("failed to push: %v", err)
		}
	}

	return cmd
}
