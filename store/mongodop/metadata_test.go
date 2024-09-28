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
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"testing"

	"github.com/prestonvasquez/diskhop/exp/dcrypto"
	"github.com/prestonvasquez/diskhop/exp/test"
	"github.com/prestonvasquez/diskhop/store"
	"github.com/stretchr/testify/assert"
)

func Test_encryptGridFSMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     []byte // 32 bytes
		gsfMeta *gridfsMetadata
		wantErr string
	}{
		{
			name:    "no metadata",
			key:     []byte("12345678901234567890123456789012"),
			gsfMeta: &gridfsMetadata{},
			wantErr: "",
		},
		{
			name: "one tag",
			key:  []byte("12345678901234567890123456789012"),
			gsfMeta: &gridfsMetadata{
				Diskhop: store.Metadata{
					Tags: []string{"tag1"},
				},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ivMgr := &test.MockIVManager{}

			key, _ := hex.DecodeString("6368616e676520746869732070617373776f726420746f206120736563726574")

			block, err := aes.NewCipher(key)
			if err != nil {
				panic(err.Error())
			}

			aesgcm, err := cipher.NewGCM(block)
			if err != nil {
				panic(err.Error())
			}

			so := dcrypto.NewAEAD(ivMgr, aesgcm)

			encBytes, err := encryptGridFSMetadata(context.Background(), so, tt.gsfMeta)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr)

				return
			} else {
				assert.Empty(t, tt.wantErr)
			}

			got, err := decryptGridFSMetadata(context.Background(), so, encBytes)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr)

				return
			} else {
				assert.Empty(t, tt.wantErr)
			}

			assert.Equal(t, tt.gsfMeta, got)
		})
	}
}
