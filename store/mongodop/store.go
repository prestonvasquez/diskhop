//
// Copyright 2024 Preston Vasquez
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
	"crypto/rand"
	"fmt"
	"io"
	"math/big"

	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"github.com/prestonvasquez/diskhop/internal/filter"
	"github.com/prestonvasquez/diskhop/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// DefaultBucketName is the default name for a bucket in MongoDB.
	DefaultBucketName         = "diskhop"
	DefaultDBName             = "diskhop"
	DefaultNameCollectionName = "name"
)

// Store is a MongoDB database for pushing and pulling data from local disk.
type Store struct {
	Pusher
	bucket      *gridfs.Bucket
	bucketName  string
	fileColl    *mongo.Collection
	commitsColl *mongo.Collection
	ivPusher    *IVPusher
	nameIndex   *nameIndex
	commits     []*store.Commit
}

var (
	_ store.Puller            = &Store{}
	_ store.Pusher            = &Store{}
	_ dcrypto.IVManagerGetter = &Store{}
	_ store.Closer            = &Store{}
	_ store.Commiter          = &Store{}
	_ store.Reverter          = &Store{}
)

// Connect will establish a connection to a MongoDB database.
func Connect(ctx context.Context, connStr, db, bucketName string) (*Store, error) {
	opts := options.Client().ApplyURI(connStr)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the MongoDB server to ensure the connection is established.
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB server: %w", err)
	}

	bucket, err := gridfs.NewBucket(
		client.Database(db),
		options.GridFSBucket().SetName(bucketName))
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	ivPusher := &IVPusher{coll: client.Database(db).Collection("initvectors")}

	fileColl := client.Database(db).Collection(bucketName + "." + "files")
	nameColl := client.Database(db).Collection(DefaultNameCollectionName)
	commitsColl := client.Database(db).Collection("commits")

	nameIndex := &nameIndex{coll: fileColl, nameColl: nameColl}

	mongoStore := &Store{
		Pusher: Pusher{
			nameIndex: nameIndex,
			bucket:    bucket,
		},
		bucket:      bucket,
		bucketName:  bucketName,
		commitsColl: commitsColl,
		ivPusher:    ivPusher,
		nameIndex:   nameIndex,
	}

	return mongoStore, nil
}

func randomSubset(files []gridfs.File, size int) ([]gridfs.File, error) {
	if size >= len(files) {
		return files, nil
	}

	chosen := make([]gridfs.File, 0, size)
	usedIndices := make(map[int]struct{})

	for len(chosen) < size {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(files))))
		if err != nil {
			return nil, fmt.Errorf("failed to generate random number: %w", err)
		}

		index := int(n.Int64())
		if _, ok := usedIndices[index]; !ok {
			usedIndices[index] = struct{}{}
			chosen = append(chosen, files[index])
		}
	}

	return chosen, nil
}

func findFiles(
	ctx context.Context,
	nidx *nameIndex,
	bucket *gridfs.Bucket,
	opts store.PullOptions,
) ([]gridfs.File, error) {
	docs := make([]filter.Document, 0, len(nidx.nameToDoc))
	for decryptedFileName, file := range nidx.nameToDoc {
		_, gfsMeta, _ := nidx.nameDoc.get(decryptedFileName)

		docs = append(docs, filter.Document{
			EncodedName: file.Name,
			Name:        decryptedFileName,
			Tags:        gfsMeta.Diskhop.Tags,
			Size:        file.Length,
		})
	}

	filteredDocs, err := filter.FilterDocuments(opts.Filter, docs)
	if err != nil {
		return nil, fmt.Errorf("failed to filter documents: %w", err)
	}

	filteredNames := make([]string, 0, len(docs))
	for _, doc := range filteredDocs {
		filteredNames = append(filteredNames, doc.EncodedName)
	}

	if len(filteredNames) == 0 && opts.Filter != "" {
		return nil, nil
	}

	filter := bson.D{}
	if len(filteredNames) > 0 {
		filter = bson.D{{Key: "filename", Value: bson.D{{Key: "$in", Value: filteredNames}}}}
	}

	cur, err := bucket.Find(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}

	gfiles := []gridfs.File{}
	for cur.Next(ctx) {
		f := gridfs.File{}
		if err := cur.Decode(&f); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}

		gfiles = append(gfiles, f)
	}

	sampleSize := opts.SampleSize
	if sampleSize == 0 {
		sampleSize = store.DefaultSampleSize
	}

	chosen, err := randomSubset(gfiles, sampleSize)
	// Select a random sample of files.
	if err != nil {
		return nil, fmt.Errorf("failed to select random subset of files: %w", err)
	}

	return chosen, nil
}

// Close will flush the nameIndex.
func (s *Store) Close(ctx context.Context) error {
	return nil
}

// Pull will retrieve a slice of documents from a remote host.
func (s *Store) Pull(ctx context.Context, buf store.DocumentBuffer, setters ...store.PullOption) (int, error) {
	opts := store.PullOptions{}
	for _, fn := range setters {
		fn(&opts)
	}

	if opts.SealOpener != nil {
		return s.EncryptedPull(ctx, buf, setters...)
	}

	panic("not implemented")

	return 0, nil
}

// PullEnc will retrieve a slice of encrypted documents from a remote host.
func (s *Store) EncryptedPull(
	ctx context.Context,
	buf store.DocumentBuffer,
	setters ...store.PullOption,
) (int, error) {
	opts := store.PullOptions{}
	for _, fn := range setters {
		fn(&opts)
	}

	if err := loadNameIndex(ctx, s.nameIndex, opts.SealOpener); err != nil {
		return 0, fmt.Errorf("failed to load name index: %w", err)
	}

	files, err := findFiles(ctx, s.nameIndex, s.bucket, opts)
	if err != nil {
		return 0, fmt.Errorf("failed to find files: %w", err)
	}

	count := len(files)

	go func() {
		for _, f := range files {
			stream, err := s.bucket.OpenDownloadStream(f.ID)
			if err != nil {
				buf.Send(nil, fmt.Errorf("failed to open download stream: %w", err))

				return
			}

			actualName, ok := s.nameIndex.hexName.get(f.Name)
			if !ok {
				buf.Send(nil, fmt.Errorf("ID not found for file name %s", f.Name))

				return
			}

			_, gfsMeta, ok := s.nameIndex.nameDoc.get(actualName)
			if !ok {
				s.nameIndex.nameDoc.add(actualName, &f, newGridFSMetadata(nil))
			}

			doc := &store.Document{
				Filename: actualName,
				Metadata: gfsMeta.Diskhop,
			}

			data := make([]byte, f.Length)
			if _, err := io.ReadFull(stream, data); err != nil {
				buf.Send(nil, fmt.Errorf("failed to read from stream: %w", err))

				return
			}

			// Decrypt the data.
			decData, err := opts.SealOpener.Open(ctx, data)
			if err != nil {
				buf.Send(nil, fmt.Errorf("failed to decrypt data: %w", err))
				return
			}

			doc.Data = decData

			// Send the document to the buffer
			buf.Send(doc, nil)
		}

		buf.Send(nil, io.EOF)
	}()

	return count, nil
}

func (s *Store) AddCommit(_ context.Context, commit *store.Commit) {
	commit.Namespace = s.bucketName

	s.commits = append(s.commits, commit)
}

func (s *Store) FlushCommits(ctx context.Context) error {
	if len(s.commits) == 0 {
		return nil
	}

	commits := make([]interface{}, 0, len(s.commits))
	for _, commit := range s.commits {
		commits = append(commits, commit)
	}

	_, err := s.commitsColl.InsertMany(ctx, commits)
	if err != nil {
		return fmt.Errorf("failed to insert commits: %w", err)
	}

	return nil
}

// GetIVManager will return an IVManager.
func (s *Store) GetIVManager() dcrypto.IVManager {
	return dcrypto.IVManager{IVPusher: s.ivPusher}
}

// Revert will revert the store to a previous state.
func (s *Store) Revert(ctx context.Context, sha string) error {
	// Get all of the commits with SHA and collect their "fileID".
	filter := bson.D{{Key: "sha", Value: sha}}

	commits, err := s.commitsColl.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find commits: %w", err)
	}

	fileNames := make([]string, 0)
	for commits.Next(ctx) {
		commit := store.Commit{}
		if err := commits.Decode(&commit); err != nil {
			return fmt.Errorf("failed to decode commit: %w", err)
		}

		fileNames = append(fileNames, commit.FileID)
	}

	// Get the ids from teh file names.
	cur, err := s.nameIndex.coll.Find(ctx, bson.D{{Key: "filename", Value: bson.D{{Key: "$in", Value: fileNames}}}})
	if err != nil {
		return fmt.Errorf("failed to find file names: %w", err)
	}

	fileIDs := []primitive.ObjectID{}
	for cur.Next(ctx) {
		file := struct {
			ID primitive.ObjectID `bson:"_id"`
		}{}

		if err := cur.Decode(&file); err != nil {
			return fmt.Errorf("failed to decode file: %w", err)
		}

		fileIDs = append(fileIDs, file.ID)
	}

	// TODO: this is naieve, but it will work for beta.
	for _, id := range fileIDs {
		// Delete file by ID
		err = s.bucket.Delete(id)
		if err != nil {
			return fmt.Errorf("failed to delete file by ID: %w", err)
		}
	}

	// Convert filenaes into object ids
	fnAsOIDs := make([]primitive.ObjectID, 0, len(fileNames))
	for _, name := range fileNames {
		oid, err := primitive.ObjectIDFromHex(name)
		if err != nil {
			return fmt.Errorf("failed to convert file name to object ID: %w", err)
		}

		fnAsOIDs = append(fnAsOIDs, oid)
	}

	// Delete all of the names for fileIDs
	if _, err := s.nameIndex.nameColl.DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: fnAsOIDs}}}}); err != nil {
		return fmt.Errorf("failed to delete names: %w", err)
	}

	// Delete all of the commits with the given SHA
	if _, err := s.commitsColl.DeleteMany(ctx, bson.D{{Key: "sha", Value: sha}}); err != nil {
		return fmt.Errorf("failed to delete commits: %w", err)
	}

	return nil
}

var (
	errFullPushRequired = fmt.Errorf("full push not implemented")
	errTagPushRequired  = fmt.Errorf("tag push not implemented")
)

func dataChanged(ctx context.Context, nidx *nameIndex, name string, rs io.ReadSeeker, opts store.PushOptions) (bool, error) {
	if err := loadNameIndex(ctx, nidx, opts.SealOpener); err != nil {
		return false, fmt.Errorf("failed to load name index: %w", err)
	}

	originalFile, meta, ok := nidx.nameDoc.get(name)
	if !ok {
		return false, errFullPushRequired
	}

	length, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return false, fmt.Errorf("failed to seek to end of file: %w", err)
	}

	// TODO: this is expedient for beta, but it's not a great way to check if
	// the file has changed. What if the file is the same size but the contents
	// are different?
	noDataChange := originalFile.Length-28 == length
	noTagChange := !meta.addTags(opts.Tags...)

	// If absolutely nothing has changed, do nothing.
	if noDataChange && noTagChange {
		return false, nil
	}

	// If there is just a tag change, update the metadata.
	if noDataChange {
		return true, errTagPushRequired
	}

	return true, errFullPushRequired
}