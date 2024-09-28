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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/prestonvasquez/diskhop/internal/osutil"
	"github.com/prestonvasquez/diskhop/store"
)

type FilePuller struct {
	p store.Puller

	progressCh chan struct{} // progressCh is the progress of the push.
	totalCh    chan int      // totalCh is the total progress of the push.
}

func NewFilePuller(p store.Puller) *FilePuller {
	return &FilePuller{
		p:          p,
		progressCh: make(chan struct{}),
		totalCh:    make(chan int, 1),
	}
}

func (fp *FilePuller) Pull(ctx context.Context, opts ...store.PullOption) error {
	buf := store.NewDocumentBuffer()
	defer buf.Close()

	count, err := fp.p.Pull(ctx, buf, opts...)
	if err != nil {
		return err
	}

	fp.totalCh <- count
	fp.progressCh = make(chan struct{}, count)

	defer close(fp.totalCh)
	defer close(fp.progressCh)

	for {
		doc, err := buf.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		file, err := os.Create(doc.Filename)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := file.Write(doc.Data); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		if tags := doc.Metadata.Tags; len(tags) > 0 {
			if err := osutil.SetTags(file, tags...); err != nil {
				return fmt.Errorf("failed to set tags: %w", err)
			}
		}

		// Do something with the document.
		fp.progressCh <- struct{}{}
	}

	return nil
}

func (fp *FilePuller) Progress() <-chan struct{} {
	return fp.progressCh
}

func (fp *FilePuller) Total() <-chan int {
	return fp.totalCh
}
