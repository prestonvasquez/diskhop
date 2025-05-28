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

package store

import (
	"context"

	"github.com/prestonvasquez/diskhop/exp/dcrypto"
)

const DefaultSampleSize = 5

type FileDescription struct {
	Name string
	Size int64
}

type PullDescription struct {
	Count            int
	Size             int64
	FileDescriptions []FileDescription
}

// Puller is an interface that defines the behavior of pulling a slice of
// documents from a remote host.
type Puller interface {
	// Pull will retrieve a slice of documents from a remote host.
	Pull(ctx context.Context, b DocumentBuffer, opts ...PullOption) (*PullDescription, error)
}

// PullOptions is a type for setting options for the pull operation.
type PullOptions struct {
	SampleSize        int    // The number of documents to pull.
	Filter            string // Filter string
	SealOpener        dcrypto.SealOpener
	DescribeOnly      bool
	DescribeFilesOnly bool
	Workers           int
	MaskName          bool // Use a UUID as a mask name
	Progress          chan NameProgress
}

type PullOption func(*PullOptions)

func WithPullSampleSize(size int) PullOption {
	return func(o *PullOptions) {
		o.SampleSize = size
	}
}

func WithPullFilter(filter string) PullOption {
	return func(o *PullOptions) {
		o.Filter = filter
	}
}

func WithPullSealOpener(so dcrypto.SealOpener) PullOption {
	return func(o *PullOptions) {
		o.SealOpener = so
	}
}

// WithPullDescribe will only describe the pull operation and not actually pull
// the documents.
func WithPullDescribe() PullOption {
	return func(o *PullOptions) {
		o.DescribeOnly = true
	}
}

func WithWorkers(workers int) PullOption {
	return func(o *PullOptions) {
		o.Workers = workers
	}
}

func WithMaskName() PullOption {
	return func(o *PullOptions) {
		o.MaskName = true
	}
}

func WithPullProgress(progress chan NameProgress) PullOption {
	return func(o *PullOptions) {
		o.Progress = progress
	}
}
