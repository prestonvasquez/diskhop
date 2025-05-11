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
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"github.com/prestonvasquez/diskhop/internal/filter"
	"github.com/prestonvasquez/diskhop/store"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	// DefaultBucketName is the default name for a bucket in MongoDB.
	DefaultBucketName         = "diskhop"
	DefaultDBName             = "diskhop"
	DefaultNameCollectionName = "name"
	defaultWorkers            = 1
)

// Store is a MongoDB database for pushing and pulling data from local disk.
type Store struct {
	Pusher
	bucket      *mongo.GridFSBucket
	bucketName  string
	fileColl    *mongo.Collection
	commitsColl *mongo.Collection
	ivPusher    *IVPusher
	nameIndex   *nameIndex
	commits     []*store.Commit
	client      *mongo.Client
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
func Connect(ctx context.Context, connStr, dbName, bucketName string) (*Store, error) {
	opts := options.Client().ApplyURI(connStr)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the MongoDB server to ensure the connection is established.
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB server: %w", err)
	}

	db := client.Database(dbName)

	bucket := db.GridFSBucket(options.GridFSBucket().SetName(bucketName))

	ivPusher := &IVPusher{coll: db.Collection("initvectors")}

	fileColl := db.Collection(bucketName + "." + "files")
	nameColl := db.Collection(DefaultNameCollectionName)
	commitsColl := db.Collection("commits")

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
		client:      client,
	}

	return mongoStore, nil
}

func randomSubset(files []mongo.GridFSFile, size int) ([]mongo.GridFSFile, error) {
	if size >= len(files) {
		return files, nil
	}

	chosen := make([]mongo.GridFSFile, 0, size)
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

// unmarshalFile is a temporary type used to unmarshal documents from the files collection and can be transformed into
// a File instance. This type exists to avoid adding BSON struct tags to the exported File type.
type unmarshalFile struct {
	ID         interface{} `bson:"_id"`
	Length     int64       `bson:"length"`
	ChunkSize  int32       `bson:"chunkSize"`
	UploadDate time.Time   `bson:"uploadDate"`
	Name       string      `bson:"filename"`
	Metadata   bson.Raw    `bson:"metadata"`
}

func findFiles(
	ctx context.Context,
	nidx *nameIndex,
	bucket *mongo.GridFSBucket,
	opts store.PullOptions,
) ([]mongo.GridFSFile, error) {
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

	cur, err := bucket.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}

	gfiles := []mongo.GridFSFile{}
	for cur.Next(ctx) {
		uf := unmarshalFile{}
		if err := cur.Decode(&uf); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}

		f := mongo.GridFSFile{
			ID:         uf.ID,
			Length:     uf.Length,
			ChunkSize:  uf.ChunkSize,
			UploadDate: uf.UploadDate,
			Name:       uf.Name,
			Metadata:   uf.Metadata,
		}

		gfiles = append(gfiles, f)
	}

	sampleSize := opts.SampleSize
	if sampleSize == 0 {
		sampleSize = store.DefaultSampleSize
	}

	if opts.DescribeOnly {
		sampleSize = len(gfiles)
	}

	chosen, err := randomSubset(gfiles, sampleSize)
	// Select a random sample of files.
	if err != nil {
		return nil, fmt.Errorf("failed to select random subset of files: %w", err)
	}

	// Sort the chosen files from smallest to largest to ensure that the maximum
	// number of files are downloaded in parallel, in the case that the download
	// stream is canceled prematurely.
	sort.Slice(chosen, func(i, j int) bool {
		return chosen[i].Length < chosen[j].Length
	})

	return chosen, nil
}

// Close will flush the nameIndex.
func (s *Store) Close(ctx context.Context) error {
	if err := s.client.Disconnect(ctx); err != nil {
		return err
	}

	return nil
}

// Pull will retrieve a slice of documents from a remote host.
func (s *Store) Pull(ctx context.Context, buf store.DocumentBuffer, setters ...store.PullOption) (*store.PullDescription, error) {
	opts := store.PullOptions{}
	for _, fn := range setters {
		fn(&opts)
	}

	if opts.SealOpener != nil {
		return s.EncryptedPull(ctx, buf, setters...)
	}

	panic("not implemented")

	return nil, nil
}

type errorDocument struct {
	doc store.Document
	err error
}

func encryptedPullWorker(
	ctx context.Context,
	s *Store,
	files <-chan mongo.GridFSFile,
	results chan<- errorDocument,
	opts store.PullOptions,
) {
	for file := range files {
		actualName, ok := s.nameIndex.hexName.get(file.Name)
		if !ok {
			results <- errorDocument{err: fmt.Errorf("ID not found for file name %s", file.Name)}

			return
		}

		_, gfsMeta, ok := s.nameIndex.nameDoc.get(actualName)
		if !ok {
			s.nameIndex.nameDoc.add(actualName, &file, newGridFSMetadata(nil))
		}

		docName := actualName
		if opts.MaskName {
			docName = uuid.New().String()
		}

		doc := &store.Document{
			Filename: docName,
			Metadata: gfsMeta.Diskhop,
		}

		stream, err := s.bucket.OpenDownloadStream(ctx, file.ID)
		if err != nil {
			results <- errorDocument{err: fmt.Errorf("failed to open download stream: %w", err)}

			return
		}

		data := make([]byte, file.Length)
		if _, err := io.ReadFull(stream, data); err != nil {
			results <- errorDocument{err: fmt.Errorf("failed to read from stream: %w", err)}

			return
		}

		// Decrypt the data.
		decData, err := opts.SealOpener.Open(ctx, data)
		if err != nil {
			results <- errorDocument{err: fmt.Errorf("failed to decrypt data: %w", err)}

			return
		}

		doc.Data = decData

		results <- errorDocument{doc: *doc}
	}
}

// PullEnc will retrieve a slice of encrypted documents from a remote host.
func (s *Store) EncryptedPull(
	ctx context.Context,
	buf store.DocumentBuffer,
	setters ...store.PullOption,
) (*store.PullDescription, error) {
	opts := store.PullOptions{}
	for _, fn := range setters {
		fn(&opts)
	}

	if err := loadNameIndex(ctx, s.nameIndex, opts.SealOpener); err != nil {
		return nil, fmt.Errorf("failed to load name index: %w", err)
	}

	files, err := findFiles(ctx, s.nameIndex, s.bucket, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find files: %w", err)
	}

	count := len(files)

	desc := &store.PullDescription{Count: count}

	go func() {
		if opts.DescribeOnly {
			return
		}

		filesCh := make(chan mongo.GridFSFile, count)
		results := make(chan errorDocument, count)

		workerCount := opts.Workers
		if workerCount == 0 {
			workerCount = defaultWorkers
		}

		for w := 0; w < workerCount; w++ {
			go encryptedPullWorker(ctx, s, filesCh, results, opts)
		}

		for i := 0; i < count; i++ {
			filesCh <- files[i]
		}
		close(filesCh)

		for a := 0; a < count; a++ {
			errDoc := <-results
			if errDoc.err != nil {
				buf.Send(nil, errDoc.err)

				continue
			}

			buf.Send(&errDoc.doc, nil)
		}

		buf.Send(nil, io.EOF)
	}()

	return desc, nil
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

	fileIDs := []bson.ObjectID{}
	for cur.Next(ctx) {
		file := struct {
			ID bson.ObjectID `bson:"_id"`
		}{}

		if err := cur.Decode(&file); err != nil {
			return fmt.Errorf("failed to decode file: %w", err)
		}

		fileIDs = append(fileIDs, file.ID)
	}

	// TODO: this is naieve, but it will work for beta.
	for _, id := range fileIDs {
		// Delete file by ID
		err = s.bucket.Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete file by ID: %w", err)
		}
	}

	// Convert filenaes into object ids
	fnAsOIDs := make([]bson.ObjectID, 0, len(fileNames))
	for _, name := range fileNames {
		oid, err := bson.ObjectIDFromHex(name)
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
