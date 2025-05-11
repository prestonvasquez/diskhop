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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// IVPusher is a struct that will push an initialization vector to the store.
type IVPusher struct {
	coll *mongo.Collection
}

var _ dcrypto.IVPusher = &IVPusher{}

// Exists will check if an initialization vector exists in the store.
func (ivp *IVPusher) Exists(ctx context.Context, iv []byte) (bool, error) {
	cur, err := ivp.coll.Find(ctx, bson.D{{Key: "ivector", Value: iv}})
	if err != nil {
		return false, fmt.Errorf("failed to find initialization vector: %w", err)
	}

	for cur.Next(ctx) {
		return true, nil
	}

	return false, nil
}

// Push will push an initialization vector to the store.
func (ivp *IVPusher) Push(ctx context.Context, iv []byte) error {
	if _, err := ivp.coll.InsertOne(ctx, bson.D{{Key: "ivector", Value: iv}}); err != nil {
		return fmt.Errorf("failed to push initialization vector: %w", err)
	}

	return nil
}
