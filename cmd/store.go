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
	"context"
	"fmt"

	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"github.com/prestonvasquez/diskhop/store"
	"github.com/prestonvasquez/diskhop/store/mongodop"
)

type diskhopStore struct {
	pusher   store.Pusher
	puller   store.Puller
	reverter store.Reverter
	ivMgr    dcrypto.IVManagerGetter
}

func newDiskhopStore(ctx context.Context, cfg config) (*diskhopStore, error) {
	switch getStoreType(cfg) {
	case storeTypeMongo:
		return newMongoStore(ctx, cfg)
	default:
		return nil, fmt.Errorf("unknown store type")
	}
}

func newMongoStore(ctx context.Context, cfg config) (*diskhopStore, error) {
	db := cfg.DB
	if db == "" {
		db = mongodop.DefaultDBName
	}

	mdb, err := mongodop.Connect(ctx, cfg.ConnString, db, cfg.CurrentBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to store: %w", err)
	}

	diskhopStore := &diskhopStore{
		pusher:   mdb,
		reverter: mdb,
		puller:   mdb,
		ivMgr:    mdb,
	}

	return diskhopStore, nil
}

func newDiskhopStoreUpstream(ctx context.Context, upstreamName string, cfg config) (*diskhopStore, error) {
	switch getStoreType(cfg) {
	case storeTypeMongo:
		return newMongoStoreUpstream(ctx, upstreamName, cfg)
	default:
		return nil, fmt.Errorf("unknown store type")
	}
}

func newMongoStoreUpstream(ctx context.Context, upstreamName string, cfg config) (*diskhopStore, error) {
	up, err := extractName(upstreamName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract upstream name: %w", err)
	}

	db := cfg.DB
	if db == "" {
		db = mongodop.DefaultDBName
	}

	mdb, err := mongodop.ConnectMigrator(ctx, cfg.ConnString, db, cfg.CurrentBranch, up)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to store: %w", err)
	}

	mdbc, err := mongodop.Connect(ctx, cfg.ConnString, db, cfg.CurrentBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to store: %w", err)
	}

	diskhopStore := &diskhopStore{
		pusher: mdb,
		ivMgr:  mdbc,
	}

	return diskhopStore, nil
}
