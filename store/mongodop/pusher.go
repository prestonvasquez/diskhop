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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/prestonvasquez/diskhop/store"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Pusher struct {
	bucket    *mongo.GridFSBucket
	nameIndex *nameIndex
}

var _ store.Pusher = &Pusher{}

var transientErrorCodes = []int{
	133, // FailedToSatisfyReadPreference
}

//// Attempt a push upload 3 times if
//const maxUploadRetries = 3

// Push pushes an object to the store.
func (p *Pusher) Push(ctx context.Context, name string, r io.ReadSeeker, opts ...store.PushOption) (string, error) {
	mergedOpts := store.PushOptions{}
	for _, fn := range opts {
		fn(&mergedOpts)
	}

	// If the seal opener is set, push an encrypted object.
	if mergedOpts.SealOpener != nil {
		return p.pushEncrypted(ctx, name, r, mergedOpts)
	}

	panic("not implemented")

	return "", nil
}

// pushEncryptedTagChange pushes an encrypted object with a tag change.
func (p *Pusher) pushEncryptedTagChange(
	ctx context.Context,
	originalFile *mongo.GridFSFile,
	meta *gridfsMetadata,
	r io.ReadSeeker,
	opts store.PushOptions,
) (string, error) {
	if err := loadNameIndex(ctx, p.nameIndex, opts.SealOpener); err != nil {
		return "", fmt.Errorf("failed to load name index: %w", err)
	}

	// Encrypt the metadata.
	encGfsMeta, err := encryptGridFSMetadata(ctx, opts.SealOpener, meta)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt metadata: %w", err)
	}

	// Update the metadata.
	updateOptions := options.UpdateOne().SetUpsert(true)
	updateDoc := bson.D{{Key: "$set", Value: bson.D{{Key: "metadata", Value: encGfsMeta}}}}

	filter := bson.D{{Key: "filename", Value: originalFile.Name}}
	if _, err = p.nameIndex.coll.UpdateOne(ctx, filter, updateDoc, updateOptions); err != nil {
		return "", fmt.Errorf("failed to update metadata: %w", err)
	}

	return originalFile.ID.(bson.ObjectID).Hex(), nil
}

// encryptedExistsPush pushes an encrypted object that already exists in the
// bucket.
func (p *Pusher) pushEncryptedChange(
	ctx context.Context,
	originalFile *mongo.GridFSFile,
	meta *gridfsMetadata,
	r io.ReadSeeker,
	opts store.PushOptions,
) (string, error) {
	if err := loadNameIndex(ctx, p.nameIndex, opts.SealOpener); err != nil {
		return "", fmt.Errorf("failed to load name index: %w", err)
	}

	length, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return "", fmt.Errorf("failed to seek to end of file: %w", err)
	}

	// TODO: this is expedient for beta, but it's not a great way to check if
	// the file has changed. What if the file is the same size but the contents
	// are different?
	noDataChange := originalFile.Length-28 == length
	noTagChange := !meta.addTags(opts.Tags...)

	// If absolutely nothing has changed, do nothing.
	if noDataChange && noTagChange {
		return originalFile.ID.(bson.ObjectID).Hex(), nil
	}

	// If there is just a tag change, update the metadata.
	if noDataChange {
		return p.pushEncryptedTagChange(ctx, originalFile, meta, r, opts)
	}

	return "", errFullPushRequired
}

// encryptedPush is a helper function that pushes an encrypted object.
func (p *Pusher) pushEncrypted(
	ctx context.Context,
	name string,
	r io.ReadSeeker,
	opts store.PushOptions,
) (string, error) {
	if err := loadNameIndex(ctx, p.nameIndex, opts.SealOpener); err != nil {
		return "", fmt.Errorf("failed to load name index: %w", err)
	}

	originalFile, meta, ok := p.nameIndex.nameDoc.get(name)

	newMeta := meta == nil
	if newMeta {
		meta = newGridFSMetadata(opts.Tags)
	} else {
		// If the metadata already exists, remove the tags
		meta.Diskhop.Tags = nil
	}

	if newMeta {
		p.nameIndex.nameToMetadata[name] = meta
	}

	if ok {
		if fileID, err := p.pushEncryptedChange(ctx, originalFile, meta, r, opts); !errors.Is(err, errFullPushRequired) {
			return fileID, err
		}

		// The change is too complex to do a partial update. Seek back to the
		// beginning of the file and re-upload the entire file.
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("failed to seek to start of file: %w", err)
		}
	} else {
		meta.addTags(opts.Tags...)
	}

	// Read and seal the bytes.
	byts, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	ciphertext, err := opts.SealOpener.Seal(ctx, byts)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt file: %w", err)
	}

	// Add new tags and encrypt the metadata.
	encryptedMeta, err := encryptGridFSMetadata(ctx, opts.SealOpener, meta)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt metadata: %w", err)
	}

	var (
		newObjectID = bson.NewObjectID()
		gridFSOpts  = options.GridFSUpload()
	)

	if len(encryptedMeta) > 0 {
		gridFSOpts.SetMetadata(encryptedMeta)
	}

	maxRetries := opts.RetryPolicy.MaxRetries
	if maxRetries == 0 {
		maxRetries = 1
	}

	var id bson.ObjectID

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			// rewind and back off
			time.Sleep(1 * time.Second)
		}

		id, err = p.bucket.UploadFromStream(ctx, newObjectID.Hex(), bytes.NewReader(ciphertext), gridFSOpts)
		if err == nil {
			break
		}

		// check for Mongo transient codes
		var srvErr mongo.ServerError
		if errors.As(err, &srvErr) {
			retryable := false
			for _, code := range transientErrorCodes {
				if srvErr.HasErrorCode(code) {
					log.Printf("Transient error code %d encountered, retrying upload for %q\n", code, name)
					retryable = attempt < maxRetries
					break
				}
			}
			if retryable {
				continue
			}
		}

		// non-transient or no retries left
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	if originalFile == nil {
		originalFile = &mongo.GridFSFile{}
	}

	p.nameIndex.nameDoc.add(name, &mongo.GridFSFile{ID: id, Name: newObjectID.Hex(), Length: int64(len(byts))}, meta)
	p.nameIndex.hexName.add(newObjectID.Hex(), name)

	newIDAsHex := newObjectID.Hex()

	// If the original file exists at this point, it's a duplicate and we
	// should delete it.
	if pid, _ := originalFile.ID.(bson.ObjectID); !pid.IsZero() {
		if err := p.bucket.Delete(ctx, pid); err != nil && !errors.Is(err, mongo.ErrFileNotFound) {
			return newIDAsHex, fmt.Errorf("failed to remove the old data with id %q from bucket: %w", pid, err)
		}
	}

	if originalFile.Name != "" {
		originalObjectID, err := bson.ObjectIDFromHex(originalFile.Name)
		if err != nil {
			return newIDAsHex, fmt.Errorf("failed to convert original name to object ID: %w", err)
		}

		if _, err := p.nameIndex.coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: originalObjectID}}); err != nil {
			return newIDAsHex, fmt.Errorf("failed to delete old file: %w", err)
		}
	}

	// Encrypt the file name.
	encFileName, err := opts.SealOpener.Seal(ctx, []byte(name))
	if err != nil {
		return newIDAsHex, fmt.Errorf("failed to encrypt file name: %w", err)
	}

	// Insert the encrypted file name into the name collection.
	idoc := bson.D{{Key: "_id", Value: newObjectID}, {Key: "data", Value: encFileName}}
	if _, err := p.nameIndex.nameColl.InsertOne(ctx, idoc); err != nil {
		return newIDAsHex, fmt.Errorf("failed to insert encrypted file name into name collection: %w", err)
	}

	return newIDAsHex, nil
}
