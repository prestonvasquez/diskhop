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
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prestonvasquez/diskhop"
	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"github.com/prestonvasquez/diskhop/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const tmpDirRoot = "tmp"

type TestStore struct {
	Pusher   store.Pusher
	Puller   store.Puller
	Reverter store.Reverter
	Commiter store.Commiter
	Mgr      dcrypto.IVManagerGetter

	// Setup and Teardown functions for the test store.
	Setup    func(*testing.T)
	Teardown func(*testing.T, context.Context)
}

type migratorKey struct {
	src, target string
}

type T struct {
	Dir             string
	NewTestStore    func(t *testing.T, ctx context.Context, bucketName string) *TestStore
	NewTestMigrator func(t *testing.T, ctx context.Context, srcBucketName, targetBucketName string) *TestStore
	Setup           func(t *testing.T, ctx context.Context)

	buckets   map[string]*TestStore
	migrators map[migratorKey]*TestStore
}

type fileData struct {
	FileName string   `yaml:"name"`
	Data     string   `yaml:"data"`
	Tags     []string `yaml:"tags"`
}

type filters struct{}

type operation struct {
	Action          string
	Args            []map[string]any
	Cipher          string
	Bucket          string
	MigrationSrc    string `yaml:"migrationSrc"`
	MigrationTarget string `yaml:"migrationTarget"`

	sealerOpener dcrypto.SealOpener
}

type testCase struct {
	Name       string
	Operations []operation
	Want       []fileData
	Cipher     string
}

type testMatrix struct {
	Cipher string
	Cases  []testCase
}

var key = make([]byte, 32)

func createTmpDir(t *testing.T) (string, func()) {
	// Get the working directory
	wd, err := os.Getwd()
	require.NoError(t, err, "failed to get working directory")

	dir := filepath.Join(wd, uuid.NewString())

	err = os.RemoveAll(dir)
	assert.NoError(t, err, "failed to remove temporary directory")

	// Remake the tmp directory
	err = os.Mkdir(dir, 0o755)
	assert.NoError(t, err, "failed to make temporary directory")

	return dir, func() { os.RemoveAll(dir) }
}

func newDCryptoAEAD(t *testing.T, mgr dcrypto.IVManagerGetter) *dcrypto.AEAD {
	key, _ := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")

	block, err := aes.NewCipher(key)
	require.NoError(t, err, "failed to create new AES cipher")

	aesgcm, err := cipher.NewGCM(block)
	require.NoError(t, err, "failed to create GCM cipher")

	return dcrypto.NewAEAD(mgr, aesgcm)
}

type pushArgs struct {
	name string
	data io.ReadSeeker
	tags []string
	sha  string
}

func newPushArgs(args map[string]any) pushArgs {
	pushArgs := pushArgs{}
	for key, value := range args {
		switch key {
		case "name":
			pushArgs.name = value.(string)
		case "sha":
			pushArgs.sha = value.(string)
		case "data":
			pushArgs.data = strings.NewReader(value.(string))
		case "tags":
			tags := value.([]any)

			pushArgs.tags = make([]string, 0, len(tags))
			for _, tag := range tags {
				pushArgs.tags = append(pushArgs.tags, tag.(string))
			}
		}
	}

	return pushArgs
}

func runPushOperation(t *testing.T, client *TestStore, op operation, dir string) {
	t.Helper()

	// If there are no args, we should do a path-level push.
	if len(op.Args) == 0 {
		fp := diskhop.NewFilePusher(client.Pusher)

		// Get the files in the directory.
		f, err := os.Open(dir)
		require.NoError(t, err, "failed to open directory")

		defer f.Close()

		pushOpts := []store.PushOption{}
		if op.sealerOpener != nil {
			pushOpts = append(pushOpts, store.WithPushSealOpener(op.sealerOpener))
		}

		err = fp.Push(context.Background(), f, pushOpts...)
		require.NoError(t, err, "failed to push encrypted file")

		return
	}

	// If there are args, we should treat it as seeding the bucket.
	for _, args := range op.Args {
		pushArgs := newPushArgs(args)

		opts := []store.PushOption{}
		if op.sealerOpener != nil {
			opts = append(opts, store.WithPushSealOpener(op.sealerOpener))
		}

		opts = append(opts, store.WithPushTags(pushArgs.tags...))

		filepath := filepath.Join(dir, pushArgs.name)

		fileID, err := client.Pusher.Push(context.Background(), filepath, pushArgs.data, opts...)
		require.NoError(t, err) // TODO: add to case to allow for expected errors

		// If a commiter is defined, then we should commit.
		if client.Commiter != nil && pushArgs.sha != "" {
			client.Commiter.AddCommit(context.Background(), &store.Commit{
				SHA:    pushArgs.sha,
				FileID: fileID,
			})
		}
	}

	if client.Commiter != nil {
		err := client.Commiter.FlushCommits(context.Background())
		require.NoError(t, err, "failed to flush commits")
	}
}

func runPullOperation(t *testing.T, client *TestStore, op operation) {
	t.Helper()

	options := []store.PullOption{}
	for _, arg := range op.Args {
		for key, value := range arg {
			switch key {
			case "filter":
				options = append(options, store.WithPullFilter(value.(string)))
			}
		}
	}

	if op.sealerOpener != nil {
		options = append(options, store.WithPullSealOpener(op.sealerOpener))
	}

	fp := diskhop.NewFilePuller(client.Puller)

	_, err := fp.Pull(context.Background(), options...)
	require.NoError(t, err, "failed to pull file")
}

type revertArgs struct {
	shas []string
}

func newRevertArgs(t *testing.T, args []map[string]any) revertArgs {
	t.Helper()

	// If there are more than  args, we should error.
	if len(args) != 1 {
		t.Fatalf("expected 1 arg for reverting, got %d", len(args))
	}

	revertArgs := revertArgs{}
	for key, value := range args[0] {
		switch key {
		case "shas":
			shas := value.([]any)

			revertArgs.shas = make([]string, 0, len(shas))
			for _, sha := range shas {
				revertArgs.shas = append(revertArgs.shas, sha.(string))
			}
		}
	}

	return revertArgs
}

func runRevertOperation(t *testing.T, client *TestStore, op operation) {
	t.Helper()

	if client.Reverter == nil {
		t.Skip("revert operation not supported")
	}

	args := newRevertArgs(t, op.Args)

	for _, sha := range args.shas {
		err := client.Reverter.Revert(context.Background(), sha)
		require.NoError(t, err, "failed to revert")
	}
}

type migrationArgs struct {
	fileName string
	tags     []string
	filter   string
}

func newMigrationArgs(t *testing.T, args []map[string]any) migrationArgs {
	t.Helper()

	// If there are more than  args, we should error.
	if len(args) != 1 {
		t.Fatalf("expected 1 arg for reverting, got %d", len(args))
	}

	migrationArgs := migrationArgs{}
	for key, value := range args[0] {
		switch key {
		case "name":
			migrationArgs.fileName = value.(string)
		case "tags":
			for _, tag := range value.([]any) {
				migrationArgs.tags = append(migrationArgs.tags, tag.(string))
			}
		case "filter":
			migrationArgs.filter = value.(string)
		}
	}

	return migrationArgs
}

func runMigrateOperation(t *testing.T, test T, op operation, dir string) {
	t.Helper()

	if test.NewTestMigrator == nil {
		t.Skip("migrate operation not supported")
	}

	require.NotEmpty(t, op.MigrationSrc, "migration source is required")
	require.NotEmpty(t, op.MigrationTarget, "migration target is required")

	key := migratorKey{
		src:    op.MigrationSrc,
		target: op.MigrationTarget,
	}

	client, ok := test.migrators[key]
	if !ok {
		client = test.NewTestMigrator(t, context.Background(), op.MigrationSrc, op.MigrationTarget)
		test.migrators[key] = client

		client.Setup(t)
	}

	args := newMigrationArgs(t, op.Args)

	opts := []store.PushOption{}

	if op.sealerOpener != nil {
		opts = append(opts, store.WithPushSealOpener(op.sealerOpener))
	}

	if len(args.tags) > 0 {
		opts = append(opts, store.WithPushTags(args.tags...))
	}

	if args.filter != "" {
		opts = append(opts, store.WithPushFilter(args.filter))
	}

	var file *os.File
	var fileName string

	// Open the file to be migrated.
	if args.fileName != "" {
		var err error

		file, err := os.Open(filepath.Join(dir, args.fileName))
		require.NoError(t, err, "failed to open file")

		fileName = file.Name()
	}

	_, err := client.Pusher.Push(context.Background(), fileName, file, opts...)
	require.NoError(t, err, "failed to migrate file")

	dirL, err := os.Open(dir)
	require.NoError(t, err, "failed to open directory")

	// Clear the directory
	//
	// Read the directory contents
	entities, err := dirL.Readdir(-1)
	require.NoError(t, err, "failed to read directory contents")

	// Remove the files from the directory.
	for _, entry := range entities {
		// Don't remove hidden files
		if entry.Name()[0] == '.' {
			continue
		}

		err := os.Remove(filepath.Join(dir, entry.Name()))
		require.NoError(t, err, "failed to remove file")
	}
}

func runTestCase(t *testing.T, test T, tc testCase) {
	t.Helper()

	test.buckets = make(map[string]*TestStore)
	test.migrators = make(map[migratorKey]*TestStore)

	test.Setup(t, context.Background())

	const defaultBucketName = "primaryTestBucket"

	// Remove old tmp dir and create a new one.
	dir, tmpTeardown := createTmpDir(t)
	defer tmpTeardown()

	// Run the operations
	for _, op := range tc.Operations {
		if op.Cipher == "" {
			op.Cipher = tc.Cipher
		}

		bucket := defaultBucketName
		if op.Bucket != "" {
			bucket = op.Bucket
		}

		client, ok := test.buckets[bucket]
		if !ok {
			client = test.NewTestStore(t, context.Background(), bucket)
			test.buckets[bucket] = client

			client.Setup(t)
		}
		switch op.Cipher {

		case "aes-gcm":
			op.sealerOpener = newDCryptoAEAD(t, client.Mgr)
		case "":
		default:
			t.Fatalf("unknown cipher: %s", op.Cipher)
		}

		switch op.Action {
		case "push":
			fmt.Println(1)
			runPushOperation(t, client, op, dir)
		case "pull":
			fmt.Println(2)
			runPullOperation(t, client, op)
		case "revert":
			runRevertOperation(t, client, op)
		case "migrate":
			runMigrateOperation(t, test, op, dir)
		default:
			t.Fatalf("unknown operation: %s", op.Action)
		}
	}

	// Read all of the files in the tmp dir into []fileData
	got := make([]fileData, 0)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "failed to read directory")

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		file, err := os.Open(filepath.Join(dir, entry.Name()))
		assert.NoError(t, err, "failed to open file")

		tags, err := diskhop.GetTags(file)
		assert.NoError(t, err, "failed to get tags")

		// Read the data from the file into a string.
		var content string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			content += scanner.Text()
		}

		err = scanner.Err()
		assert.NoError(t, err, "failed to read file")

		got = append(got, fileData{
			FileName: filepath.Base(file.Name()),
			Tags:     tags,
			Data:     content,
		})

		file.Close()
	}

	assert.ElementsMatch(t, tc.Want, got)

	for _, client := range test.buckets {
		client.Teardown(t, context.Background())
	}
}

func runTestMatrix(t *testing.T, test T, file os.DirEntry) {
	t.Helper()

	filePath := filepath.Join(test.Dir, file.Name())

	// Read the file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading file %s: %v", filePath, err)
	}

	// Unmarshal the YAML content into the Config struct
	testMatrix := testMatrix{}

	err = yaml.Unmarshal(content, &testMatrix)
	require.NoError(t, err, "failed to unmarshal test data")

	for _, tc := range testMatrix.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Cipher == "" {
				tc.Cipher = testMatrix.Cipher
			}

			runTestCase(t, test, tc)
		})
	}
}

func Run(t *testing.T, test T) {
	t.Helper()

	files, err := os.ReadDir(test.Dir)
	require.NoError(t, err, "failed to read test data")

	// Iterate through the entries of the directory
	for _, file := range files {
		if file.IsDir() {
			t.Skipf("skipping directory: %s", file.Name())
		}

		t.Run(file.Name(), func(t *testing.T) {
			runTestMatrix(t, test, file)
		})
	}
}
