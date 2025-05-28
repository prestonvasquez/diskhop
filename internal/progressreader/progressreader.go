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

package progressreader

import (
	"io"

	"github.com/prestonvasquez/diskhop/store"
)

// Reader wraps an io.Reader and reports progress as percentage via a channel.
type Reader struct {
	r       io.Reader
	read    int64
	total   int64
	updates chan store.NameProgress // send progress percentage (0–100) here
	name    string
}

var _ io.Reader = (*Reader)(nil)

// NewReader wraps r and reports progress to updates channel.
// Caller is responsible for closing the channel if needed.
func NewReader(r io.Reader, total int64, name string, updates chan store.NameProgress) *Reader {
	return &Reader{
		name:    name,
		r:       r,
		total:   total,
		updates: updates,
	}
}

func (p *Reader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.read += int64(n)
	if p.total > 0 && p.updates != nil {
		select {
		case p.updates <- store.NameProgress{Name: p.name, Progress: float64(p.read) / float64(p.total) * 100}:
		default:
			// non-blocking send — drop update if no listener
		}
	}
	return n, err
}

func (p *Reader) Close() error {
	if closer, ok := p.r.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
