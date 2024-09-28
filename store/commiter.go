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
	"crypto/sha1"
	"encoding/hex"

	"github.com/google/uuid"
)

type Commit struct {
	SHA       string `json:"uuid",bson:"uuid"`
	Namespace string `json:"namespace",bson:"namespace"`
	FileID    string `json:"fileId",bson:"fileId"`
}

// Commiter is an interface that defines the behavior of committing.
type Commiter interface {
	AddCommit(context.Context, *Commit)
	FlushCommits(context.Context) error
}

// NewSHA generates a new SHA-1 hash based on a name.
func NewSHA(name string) string {
	// Generate a new UUID
	newUUID := uuid.New()

	// Convert UUID to string
	uuidString := newUUID.String()

	// Concatenate the base string and the UUID
	data := name + uuidString

	// Create a new SHA-1 hash
	hash := sha1.New()

	// Write the concatenated data to the hash
	hash.Write([]byte(data))

	// Compute the final hash (as a slice of bytes)
	hashBytes := hash.Sum(nil)

	// Convert the hash to a hexadecimal string
	return hex.EncodeToString(hashBytes)
}
