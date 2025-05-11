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
	"io"

	"github.com/prestonvasquez/diskhop/exp/dcrypto"
)

// Pusher is an interface that defines the behavior of pushing.
type Pusher interface {
	Push(ctx context.Context, name string, r io.ReadSeeker, opts ...PushOption) (string, error)
}

type PushOption func(*PushOptions)

// PushOptions defines the options for pushing an object.
type PushOptions struct {
	Tags        []string // Metadata tags to associate with the object.
	SealOpener  dcrypto.SealOpener
	Filter      string // Filter string
	RetryPolicy RetryPolicy
}

// WithPushTags sets the tags for the object.
func WithPushTags(tags ...string) PushOption {
	return func(o *PushOptions) {
		o.Tags = tags
	}
}

// WithPushSealOpener sets the sealer and opener for the object for encryption.
func WithPushSealOpener(so dcrypto.SealOpener) PushOption {
	return func(o *PushOptions) {
		o.SealOpener = so
	}
}

// WithPushFilter will allow the user to set a filter for the push operation,
// specifically to avoid downloading chunk data for migration.
func WithPushFilter(filter string) PushOption {
	return func(o *PushOptions) {
		o.Filter = filter
	}
}

// withPushRetryPolicy sets the retry policy for the push operation.
func WithRetryPolicy(retryPolicy RetryPolicy) PushOption {
	return func(o *PushOptions) {
		o.RetryPolicy = retryPolicy
	}
}
