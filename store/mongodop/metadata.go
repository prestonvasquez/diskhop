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

package mongodop

import (
	"context"
	"fmt"

	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"github.com/prestonvasquez/diskhop/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type gridfsMetadata struct {
	Diskhop store.Metadata `bson:"diskhop"`
}

func newGridFSMetadata(tags []string) *gridfsMetadata {
	gfsMeta := &gridfsMetadata{}

	if len(tags) > 0 {
		gfsMeta.Diskhop.Tags = tags
	}

	return gfsMeta
}

func decryptGridFSMetadata(ctx context.Context, opener dcrypto.Opener, raw bson.Raw) (*gridfsMetadata, error) {
	var doc bson.M
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	diskhopMetadataBinary := doc[metadataKey].(primitive.Binary)

	decDiskhopMetdataBsonRaw := bson.Raw(diskhopMetadataBinary.Data)
	var err error

	decDiskhopMetdataBsonRaw, err = opener.Open(ctx, diskhopMetadataBinary.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt metadata: %w", err)
	}

	var metadata store.Metadata
	if err := bson.Unmarshal(decDiskhopMetdataBsonRaw, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &gridfsMetadata{Diskhop: metadata}, nil
}

func encryptGridFSMetadata(
	ctx context.Context,
	sealer dcrypto.Sealer,
	gfsMeta *gridfsMetadata,
) (bson.Raw, error) {
	metaBytes, err := bson.Marshal(gfsMeta.Diskhop)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	encMetaBytes, err := sealer.Seal(ctx, bson.Raw(metaBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt metadata: %w", err)
	}

	doc := bson.M{metadataKey: primitive.Binary{Data: encMetaBytes}}

	docBytes, err := bson.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return bson.Raw(docBytes), nil
}

func (gfsMeta *gridfsMetadata) diskhopMap() map[string]interface{} {
	if gfsMeta == nil {
		return nil
	}

	return map[string]interface{}{
		tagKey: gfsMeta.Diskhop.Tags,
	}
}

func (gfsMeta *gridfsMetadata) hasTag(tags ...string) bool {
	if gfsMeta == nil {
		return false
	}

	for _, tag := range tags {
		for _, t := range gfsMeta.Diskhop.Tags {
			if t == tag {
				return true
			}
		}
	}

	return false
}

func (gfsMeta *gridfsMetadata) hasAllTags(tags ...string) bool {
	if gfsMeta == nil || len(tags) == 0 {
		return false
	}

	tagSet := make(map[string]bool) // Use a set to check tag presence efficiently
	for _, t := range gfsMeta.Diskhop.Tags {
		tagSet[t] = true
	}

	for _, tag := range tags {
		if !tagSet[tag] { // If any tag is not found, return false
			return false
		}
	}
	return true
}

// addTags will add tags to the metadata of a gridfs file without deduplicating
// them. Returns true if the tags list was extended.
func (gfsMeta *gridfsMetadata) addTags(tags ...string) bool {
	if gfsMeta == nil {
		return false
	}

	if len(tags) == 0 && len(gfsMeta.Diskhop.Tags) == 0 {
		return false
	}

	extended := false

	dup := make(map[string]struct{})
	for _, tag := range gfsMeta.Diskhop.Tags {
		dup[tag] = struct{}{}
	}

	for _, tag := range tags {
		if _, ok := dup[tag]; ok {
			continue
		}

		gfsMeta.Diskhop.Tags = append(gfsMeta.Diskhop.Tags, tag)
		extended = true
	}

	return extended
}
