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
	"io"
	"log"

	"github.com/prestonvasquez/diskhop/store"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Migrator is a store.EncPusher that migrates files from one MongoDB gridfs
// bucket to another.
type Migrator struct {
	client           *mongo.Client
	database         string
	srcBucket        *gridfs.Bucket
	nameIndex        nameIndex
	targetBucket     *gridfs.Bucket
	srcBucketName    string
	targetBucketName string
	targetNameColl   *mongo.Collection
}

var _ store.Pusher = &Migrator{}

// ConnectMigrator connects to the MongoDB server and returns a new Migrator.
func ConnectMigrator(ctx context.Context, connStr string, db, srcB, targB string) (*Migrator, error) {
	opts := options.Client().ApplyURI(connStr)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the MongoDB server to ensure the connection is established.
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB server: %w", err)
	}

	fileColl := client.Database(db).Collection(srcB + "." + "files")
	nameColl := client.Database(db).Collection(DefaultNameCollectionName)

	srcBucket, err := gridfs.NewBucket(
		client.Database(db),
		options.GridFSBucket().SetName(srcB))
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	targetBucket, err := gridfs.NewBucket(
		client.Database(db),
		options.GridFSBucket().SetName(targB))
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	pusher := &Migrator{
		client:           client,
		database:         db,
		nameIndex:        nameIndex{coll: fileColl, nameColl: nameColl},
		srcBucket:        srcBucket,
		targetBucket:     targetBucket,
		targetBucketName: targB,
		srcBucketName:    srcB,
		targetNameColl:   client.Database(db).Collection(DefaultNameCollectionName),
	}

	return pusher, nil
}

// PushEnc migrates the file with the given name from the source bucket to the
// target bucket.
func (up *Migrator) Push(
	ctx context.Context,
	name string,
	r io.ReadSeeker,
	opts ...store.PushOption,
) (string, error) {
	mergedOpts := store.PushOptions{}
	for _, fn := range opts {
		fn(&mergedOpts)
	}

	if err := loadNameIndex(ctx, &up.nameIndex, mergedOpts.SealOpener); err != nil {
		return "", fmt.Errorf("failed to load name index: %w", err)
	}

	// Get the file id for the name.
	doc, meta, ok := up.nameIndex.nameDoc.get(name)
	if !ok {
		return "", fmt.Errorf("file not found: %s", name)
	}

	changed, err := dataChanged(ctx, &up.nameIndex, name, r, mergedOpts)
	if !changed && err == nil {
		fileID := doc.ID
		// If nothing has changed, then we use an aggregation pipeline to
		// move the data from the source to the target.
		pipeline := mongo.Pipeline{
			// Match the document
			bson.D{{"$match", bson.D{{"_id", fileID}}}},
			// Add the document to the target collection
			bson.D{{"$merge", bson.D{{"into", up.targetBucketName + "." + "files"}, {"whenMatched", "merge"}}}},
		}

		// Merge File into the target
		srcFileColl := up.client.Database(up.database).Collection(up.srcBucketName + "." + "files")

		_, err = srcFileColl.Aggregate(context.TODO(), pipeline)
		if err != nil {
			log.Fatal("Error moving file:", err)
		}

		// Merge chunks into the target
		//
		// Define the aggregation pipeline to move chunks
		chunksPipeline := mongo.Pipeline{
			// Match the chunks for the given file ID
			bson.D{{"$match", bson.D{{"files_id", fileID}}}},
			// Merge the chunks into the target collection
			bson.D{{"$merge", bson.D{{"into", up.targetBucketName + "." + "chunks"}, {"whenMatched", "merge"}}}},
		}

		srcChunksColl := up.client.Database(up.database).Collection(up.srcBucketName + "." + "chunks")

		// Execute the aggregation pipeline for the chunks
		_, err = srcChunksColl.Aggregate(context.TODO(), chunksPipeline)
		if err != nil {
			log.Fatal("Error moving chunks:", err)
		}
	} else {

		meta.addTags(mergedOpts.Tags...)

		// Add new tags and encrypt the metadata.
		encryptedMeta, err := encryptGridFSMetadata(ctx, mergedOpts.SealOpener, meta)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt metadata: %w", err)
		}

		// Download the file from source database.
		stream, err := up.srcBucket.OpenDownloadStream(doc.ID)
		if err != nil {
			return "", fmt.Errorf("failed to open download stream: %w", err)
		}

		data := make([]byte, doc.Length)
		_, err = stream.Read(data)
		if err != nil {
			return "", fmt.Errorf("failed to read data from stream: %w", err)
		}

		stream.Close()

		gfsOpts := options.GridFSUpload().SetMetadata(encryptedMeta)

		// Upload the file to target database.
		uploadStream, err := up.targetBucket.OpenUploadStream(doc.Name, gfsOpts)
		if err != nil {
			return "", fmt.Errorf("failed to open upload stream: %w", err)
		}

		_, err = uploadStream.Write(data)
		if err != nil {
			return "", fmt.Errorf("failed to write data to stream: %w", err)
		}

		uploadStream.Close()
	}

	// Delete the file from source database.
	err = up.srcBucket.Delete(doc.ID)
	if err != nil {
		return "", fmt.Errorf("failed to delete file from source bucket: %w", err)
	}

	return "", nil
}
