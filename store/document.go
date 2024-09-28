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

package store

import (
	"errors"
	"time"
)

type Metadata struct {
	Tags []string `bson:"tags,omitempty"` // Tags associated with the document
}

// Document is the data structure that is either pulled from a remote host or
// that must be constructed to push to a remote host. Note that this structure
// contains only descriptive information of the document, not the contents.
type Document struct {
	ID          []byte    // Unique identifier
	Size        int64     // Size of the document
	UploadDate  time.Time // When the document was uploaded
	Filename    string    // Name of the file
	Metadata    Metadata  // Contextual data
	ContentType string    // Type of data
	Data        []byte    // Data
}

// DocumentBuffer manages a dynamically-sized buffer of Documents.
type DocumentBuffer struct {
	ch  chan *Document
	err chan error
}

// NewDocumentBuffer creates a new DocumentBuffer.
func NewDocumentBuffer() DocumentBuffer {
	return DocumentBuffer{
		ch:  make(chan *Document),
		err: make(chan error, 1),
	}
}

// Next returns the next document and any associated error.
func (db *DocumentBuffer) Next() (*Document, error) {
	select {
	case doc, ok := <-db.ch:
		if !ok {
			return nil, errors.New("document channel closed")
		}
		return doc, nil
	case err := <-db.err:
		return nil, err
	}
}

// Send adds a document to the buffer and sends any error if encountered.
func (db *DocumentBuffer) Send(doc *Document, err error) {
	if err != nil {
		db.err <- err
	} else {
		db.ch <- doc
	}
}

func (db *DocumentBuffer) Close() {
	close(db.ch)
	close(db.err)
}
