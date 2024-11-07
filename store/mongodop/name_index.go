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
	"errors"
	"fmt"
	"regexp"

	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

const (
	tagKey      = "tags"
	metadataKey = "diskhop"
)

// hexName keeps a map of string hex to the decrypted file name.
type hexName struct {
	hexToName map[string]string // hex -> decrypted name
}

// loadHexName loads the hexName map from the database.
func loadHexName(ctx context.Context, opener dcrypto.Opener, coll *mongo.Collection) (*hexName, error) {
	hn := &hexName{
		hexToName: make(map[string]string),
	}

	cur, err := coll.Find(ctx, bson.D{})
	if errors.Is(err, mongo.ErrNilDocument) {
		return hn, nil
	}

	if err != nil {
		return nil, err
	}

	type nameDoc struct {
		ID   primitive.ObjectID `bson:"_id"`
		Data primitive.Binary
	}

	for cur.Next(ctx) {
		doc := nameDoc{}
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}

		actualName, err := opener.Open(ctx, doc.Data.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt name: %w", err)
		}

		hn.add(doc.ID.Hex(), string(actualName))
	}

	return hn, nil
}

func (hn *hexName) add(hex, name string) {
	if hn.hexToName == nil {
		hn.hexToName = make(map[string]string)
	}

	hn.hexToName[hex] = name
}

func (hn *hexName) get(hex string) (string, bool) {
	if hn.hexToName == nil {
		return "", false
	}

	return hn.hexToName[hex], true
}

// nameDoc is a map of decrypted names to documents.
type nameDoc struct {
	nameToDoc      map[string]*gridfs.File    // decrypted name -> document
	nameToMetadata map[string]*gridfsMetadata //  decrypted name -> metadata
}

// loadNameDoc loads the nameDoc map from the database.
func loadNameDoc(ctx context.Context, opener dcrypto.Opener, coll *mongo.Collection, hexName *hexName) (*nameDoc, error) {
	nd := &nameDoc{
		nameToDoc:      make(map[string]*gridfs.File),
		nameToMetadata: make(map[string]*gridfsMetadata),
	}

	cur, err := coll.Find(ctx, bson.D{})
	if errors.Is(err, mongo.ErrNilDocument) {
		return nd, nil
	}

	if err != nil {
		return nil, err
	}

	for cur.Next(ctx) {
		file := gridfs.File{}
		if err := cur.Decode(&file); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}

		fileName, ok := hexName.get(file.Name)
		if !ok {
			return nil, fmt.Errorf("ID not found for file name %s", file.Name)
		}

		metadata, _ := decryptGridFSMetadata(ctx, opener, file.Metadata)

		nd.add(fileName, &file, metadata)

	}

	return nd, nil
}

func (nd *nameDoc) add(name string, doc *gridfs.File, metadata *gridfsMetadata) {
	if nd.nameToDoc == nil {
		nd.nameToDoc = make(map[string]*gridfs.File)
		nd.nameToMetadata = make(map[string]*gridfsMetadata)
	}

	nd.nameToDoc[name] = doc
	nd.nameToMetadata[name] = metadata
}

func (nd *nameDoc) get(name string) (*gridfs.File, *gridfsMetadata, bool) {
	if nd.nameToDoc == nil {
		return nil, nil, false
	}

	doc, ok := nd.nameToDoc[name]
	if !ok {
		return nil, nil, false
	}

	meta, ok := nd.nameToMetadata[name]
	if !ok {
		return nil, nil, false
	}

	return doc, meta, true
}

// nameIndex maps names to their gridfs file id. This is specifically used to
// check if an encrypted file already exists in the store.
type nameIndex struct {
	*hexName
	*nameDoc

	coll     *mongo.Collection
	nameColl *mongo.Collection
}

func loadNameIndex(ctx context.Context, nidx *nameIndex, opener dcrypto.Opener) error {
	if nidx.hexName != nil {
		return nil
	}

	var err error

	nidx.hexName, err = loadHexName(ctx, opener, nidx.nameColl)
	if err != nil {
		return fmt.Errorf("failed to load hexName: %w", err)
	}

	if nidx.nameDoc != nil {
		return nil
	}

	nidx.nameDoc, err = loadNameDoc(ctx, opener, nidx.coll, nidx.hexName)
	if err != nil {
		return fmt.Errorf("failed to load nameDoc: %w", err)
	}

	return nil
}

// unionNames returns a list of names that match any of the given regular
// expressions.
func unionNames(nidx nameIndex, names ...string) ([]string, error) {
	nameFilter := []string{}

	for fileName, file := range nidx.nameToDoc {
		for _, filter := range names {
			// Compile the regex pattern for each filter name
			re, err := regexp.Compile(filter)
			if err != nil {
				return nil, fmt.Errorf("failed to compile regular expression: %w", err)
			}

			// If any regex matches, add the file to the nameFilter and break out of the loop
			if re.MatchString(fileName) {
				nameFilter = append(nameFilter, file.Name)
				break
			}
		}
	}

	return nameFilter, nil
}

// intersectNames returns a list of names that match all of the given regular
// expressions.
func intersectNames(nidx nameIndex, names ...string) ([]string, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("no filters provided")
	}

	nameFilter := []string{}

	// Loop through each file
	for fileName, file := range nidx.nameToDoc {
		matchAll := true
		for _, filter := range names {
			re, err := regexp.Compile(filter)
			if err != nil {
				return nil, fmt.Errorf("failed to compile regular expression: %w", err)
			}
			if !re.MatchString(fileName) {
				matchAll = false
				break
			}
		}
		if matchAll {
			nameFilter = append(nameFilter, file.Name)
		}
	}

	return nameFilter, nil
}

// filterNames returns a list of names that match the given regular expressions.
func newNamesFilter(nidx nameIndex, names []string, union bool) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	if union {
		nameFilter, err := unionNames(nidx, names...)
		if err != nil {
			return nil, fmt.Errorf("failed to union names: %w", err)
		}

		return nameFilter, nil
	}

	nameFilter, err := intersectNames(nidx, names...)
	if err != nil {
		return nil, fmt.Errorf("failed to intersect names: %w", err)
	}

	return nameFilter, nil
}

// unionTags returns a list of names that match any of the given tags.
func unionTags(nidx nameIndex, tags ...string) ([]string, error) {
	tagFilter := []string{}
	for fileName, meta := range nidx.nameToMetadata {
		if meta.hasTag(tags...) {
			file := nidx.nameToDoc[fileName]

			tagFilter = append(tagFilter, file.Name)
		}
	}

	return tagFilter, nil
}

// intersectTags returns a list of names that match all of the given tags.
func intersectTags(nidx nameIndex, tags ...string) ([]string, error) {
	if len(tags) == 0 {
		return nil, fmt.Errorf("no tags provided")
	}
	tagFilter := []string{}
	for fileName, meta := range nidx.nameToMetadata {
		if meta.hasAllTags(tags...) { // Ensure all tags are present
			file := nidx.nameToDoc[fileName]

			tagFilter = append(tagFilter, file.Name)
		}
	}
	return tagFilter, nil
}

// newTagsFilter returns a lits of filenames that match the given tags.
func newTagsFilter(nidx nameIndex, tags []string, union bool) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	if union {
		tagFilter, err := unionTags(nidx, tags...)
		if err != nil {
			return nil, fmt.Errorf("failed to union tags: %w", err)
		}

		return tagFilter, nil
	}

	tagFilter, err := intersectTags(nidx, tags...)
	if err != nil {
		return nil, fmt.Errorf("failed to intersect tags: %w", err)
	}

	return tagFilter, nil
}
