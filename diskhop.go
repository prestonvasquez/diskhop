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

package diskhop

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"github.com/prestonvasquez/diskhop/store"
)

func commit(ctx context.Context, commiter store.Commiter, msg string, fileID string) {
	if commiter == nil {
		return
	}

	sha := store.NewSHA(msg)

	commiter.AddCommit(ctx, &store.Commit{
		SHA:    sha,
		FileID: fileID,
	})
}

func flushCommits(ctx context.Context, commiter store.Commiter) error {
	if commiter == nil {
		return nil
	}

	return commiter.FlushCommits(ctx)
}

func secureDelete(filename string) error {
	// Open the file for reading and writing
	file, err := os.OpenFile(filename, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	size := stat.Size()

	// Overwrite the file with random data
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	// Create a buffer with random data
	randomData := make([]byte, size)
	if _, err := rand.Read(randomData); err != nil {
		return fmt.Errorf("failed to generate random data: %w", err)
	}

	if _, err := file.Write(randomData); err != nil {
		return fmt.Errorf("failed to write random data to file: %w", err)
	}

	// Ensure all data is flushed to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Close the file before deleting
	file.Close()

	// Remove the file
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

func Clean(entities []os.FileInfo) error {
	// Remove the files from the directory.
	for _, entry := range entities {
		// Don't remove hidden files
		if entry.Name()[0] == '.' {
			continue
		}

		if err := secureDelete(entry.Name()); err != nil {
			return fmt.Errorf("failed to securely delete file: %w", err)
		}
	}

	return nil
}
