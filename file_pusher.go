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
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/prestonvasquez/diskhop/store"
)

// FilePusher is a pusher that pushes files to the store.
type FilePusher struct {
	p store.Pusher

	ProgressTracker ProgressTracker
}

// NewFilePusher creates a new file pusher.
func NewFilePusher(p store.Pusher) *FilePusher {
	return &FilePusher{p: p}
}

func (fp *FilePusher) PushFromInfo(ctx context.Context, fi os.FileInfo, opts ...store.PushOption) (string, error) {
	filePath, err := filepath.Abs(fi.Name())
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	base := filepath.Base(filePath) // Do not read hidden files.
	if base[0] == '.' {
		return "", nil
	}

	// TODO: handle directories.
	if base == "" {
		return "", nil
	}

	// Open the file
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to open file for push: %w", err)
	}

	defer file.Close()

	tags, err := GetTags(file)
	if err != nil {
		return "", fmt.Errorf("failed to get tags for file: %w", err)
	}

	fileID, err := fp.p.Push(ctx, file.Name(), file, append(opts, store.WithPushTags(tags...))...)
	if err != nil {
		return "", fmt.Errorf("failed to push file from path: %w", err)
	}

	return fileID, nil
}

// Push will push the files in the directory to the store.
func (fp *FilePusher) Push(ctx context.Context, f *os.File, opts ...store.PushOption) error {
	commiter, ok := fp.p.(store.Commiter)
	if ok {
		defer flushCommits(ctx, commiter)
	}

	// Get the files in the directory.
	f, err := os.Open(f.Name())
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}

	defer func() { _ = f.Close() }()

	// Read the directory contents
	entities, err := f.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read directory contents: %w", err)
	}

	if len(entities) == 0 {
		return nil
	}

	var noClean bool

	defer func() {
		if noClean {
			return
		}
		if err := Clean(entities); err != nil {
			panic(err)
		}
	}()

	for _, entry := range entities {
		if entry.IsDir() {
			continue
		}

		fileID, err := fp.PushFromInfo(ctx, entry, opts...)
		if err != nil {
			noClean = true
			log.Printf("failed to push file: %s\n", err)
			//return fmt.Errorf("failed to push file: %w", err)
		}

		if commiter != nil {
			commit(ctx, commiter, "push", fileID)
		}

		if fp.ProgressTracker != nil {
			if err := fp.ProgressTracker.Add(1); err != nil {
				noClean = true
				return fmt.Errorf("failed to add to progress tracker: %w", err)
			}
		}
	}

	return nil
}
