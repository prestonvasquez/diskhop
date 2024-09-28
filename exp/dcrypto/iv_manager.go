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

package dcrypto

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// ErrNotIVManagement is an error that indicates the pusher does not support
// IV management.
var ErrNotIVManagement = errors.New("pusher does not support IV management")

// IVPusher defines methods for managing initialization vectors (IVs) for
// encryption. The IV in GCM must be unique for every encryption operation with
// the same key.
type IVPusher interface {
	Exists(ctx context.Context, iv []byte) (bool, error)
	Push(ctx context.Context, iv []byte) error
}

// IVManager is a struct that embeds the IVPusher interface. It provides a
// default implementation for managing IVs.
type IVManager struct {
	IVPusher
}

// IVManagerGetter defines a method for retrieving an IVManager instance. This
// is useful for dependency injection or when multiple implementations of
// IVPusher are used.
type IVManagerGetter interface {
	GetIVManager() IVManager
}

// generateInitializationVector will generate a new initialization vector for
// encryption. This method will also push the IV to the store.
func generateInitializationVector(ctx context.Context, ivMgr IVManagerGetter, nonceSize int) ([]byte, error) {
	ivManager := ivMgr.GetIVManager()

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to read encryption nonce: %w", err)
	}

	notOk, err := ivManager.IVPusher.Exists(ctx, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to check if IV exists: %w", err)
	}

	// If the IV already exists, then try again.
	if notOk {
		return generateInitializationVector(ctx, ivMgr, nonceSize)
	}

	if err := ivManager.IVPusher.Push(ctx, nonce); err != nil {
		return nil, fmt.Errorf("failed to push IV: %w", err)
	}

	return nonce, nil
}
