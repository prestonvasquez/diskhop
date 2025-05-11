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

package test

import (
	"context"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/prestonvasquez/diskhop/exp/test"
	"github.com/prestonvasquez/diskhop/store/mongodop"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const testdataDir = "../../../testdata"

func TestMongoE2E(t *testing.T) {
	test.Run(t, test.T{
		Dir:             testdataDir,
		NewTestStore:    newTestStore,
		NewTestMigrator: newTestMigrator,
		Setup:           setup,
	})
}

func newTestStore(t *testing.T, ctx context.Context, bucketName string) *test.TestStore {
	t.Helper()

	const database = "test"

	uri := os.Getenv("MONGODB_URI")

	// Create a connection to the test server for setup and teardown.
	clientOpts := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(clientOpts)
	require.NoError(t, err, "failed to connect to mongodb")

	// Create a connection to the test server to test diskhop behavior.
	mstore, err := mongodop.Connect(context.Background(), uri, database, bucketName)
	require.NoError(t, err, "failed to connect to mongodb store")

	return &test.TestStore{
		Pusher:   mstore,
		Puller:   mstore,
		Commiter: mstore,
		Reverter: mstore,
		Mgr:      mstore,
		Setup: func(t *testing.T) {
			t.Helper()
		},
		Teardown: func(t *testing.T, ctx context.Context) {
			t.Helper()

			err = client.Disconnect(ctx)
			require.NoError(t, err, "failed to disconnect from mongodb")
		},
	}
}

func newTestMigrator(t *testing.T, ctx context.Context, src, target string) *test.TestStore {
	t.Helper()

	const database = "test"

	uri := os.Getenv("MONGODB_URI")
	clientOpts := options.Client().ApplyURI(uri)

	// Drop the database before and after test.
	client, err := mongo.Connect(clientOpts)
	require.NoError(t, err, "failed to connect to mongodb")

	migrator, err := mongodop.ConnectMigrator(context.Background(), uri, database, src, target)
	require.NoError(t, err, "failed to connect to mongodb store")

	return &test.TestStore{
		Pusher: migrator,
		Setup: func(t *testing.T) {
			t.Helper()
		},
		Teardown: func(t *testing.T, ctx context.Context) {
			t.Helper()

			err = client.Disconnect(ctx)
			require.NoError(t, err, "failed to disconnect from mongodb")
		},
	}
}

//func setup(t *testing.T, ctx context.Context) {
//	t.Helper()
//
//	const database = "test"
//
//	uri := os.Getenv("MONGODB_URI")
//	if uri == "" {
//		mongodbContainer, err := mongodb.Run(ctx, "mongo:7.0.8")
//		require.NoError(t, err, "failed to start mongodb container")
//
//		host, err := mongodbContainer.Host(ctx)
//		require.NoError(t, err, "failed to get mongodb host")
//
//		port, err := mongodbContainer.MappedPort(ctx, "27017/tcp")
//		require.NoError(t, err, "failed to get mongodb port")
//
//		uri = (&url.URL{
//			Scheme:   "mongodb",
//			Host:     net.JoinHostPort(host, port.Port()),
//			Path:     "/",
//			RawQuery: "directConnection=true",
//		}).String()
//
//		os.Setenv("MONGODB_URI", uri)
//	}
//
//	clientOpts := options.Client().ApplyURI(uri)
//
//	// Drop the database before and after test.
//	client, err := mongo.Connect(clientOpts)
//	require.NoError(t, err, "failed to connect to mongodb")
//
//	defer func() { _ = client.Disconnect(context.Background()) }()
//
//	// Ping the server to ensure the connection is established.
//	err = client.Ping(context.Background(), nil)
//	require.NoError(t, err, "failed to ping mongodb")
//
//	err = client.Database(database).Drop(context.Background())
//	require.NoError(t, err, "failed to drop database")
//}
//

func setup(t *testing.T, ctx context.Context) {
	t.Helper()

	const database = "test"
	uri := os.Getenv("MONGODB_URI")

	if uri == "" {
		// Start Mongo with test commands enabled
		req := testcontainers.ContainerRequest{
			Image:        "mongo:7.0.8",
			ExposedPorts: []string{"27017/tcp"},
			Cmd:          []string{"mongod", "--setParameter", "enableTestCommands=1"},
			WaitingFor:   wait.ForListeningPort("27017/tcp").WithStartupTimeout(30 * time.Second),
		}
		mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		require.NoError(t, err, "failed to start mongodb container")

		host, err := mongoC.Host(ctx)
		require.NoError(t, err, "failed to get mongodb host")
		port, err := mongoC.MappedPort(ctx, "27017/tcp")
		require.NoError(t, err, "failed to get mongodb port")

		uri = (&url.URL{
			Scheme:   "mongodb",
			Host:     net.JoinHostPort(host, port.Port()),
			Path:     "/",
			RawQuery: "directConnection=true",
		}).String()
		os.Setenv("MONGODB_URI", uri)
	}

	clientOpts := options.Client().ApplyURI(uri)

	// Connect, ping, and drop the test database
	client, err := mongo.Connect(clientOpts)
	require.NoError(t, err, "failed to connect to mongodb")
	defer func() { _ = client.Disconnect(context.Background()) }()

	require.NoError(t, client.Ping(context.Background(), nil), "failed to ping mongodb")
	require.NoError(t, client.Database(database).Drop(context.Background()), "failed to drop database")
}
