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

package osutil

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetTags(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "skip-test")
	require.NoError(t, err, "failed to create temporary file")

	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// If error contains "unsupported operating system", then skip the test.
	if err := SetTags(nil); err != nil && strings.Contains(err.Error(), "unsupported operating system") {
		t.Skip("unsupported operating system")
	}

	tests := []struct {
		name    string
		tags    []string
		wantErr string
	}{
		{
			name:    "no tags",
			tags:    []string{},
			wantErr: "",
		},
		{
			name:    "nil tags",
			tags:    nil,
			wantErr: "",
		},
		{
			name:    "one tag",
			tags:    []string{"tag1"},
			wantErr: "",
		},
		{
			name:    "two tags",
			tags:    []string{"tag1", "tag2"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file to test the function
			tmpFile, err := os.CreateTemp("", "test")
			require.NoError(t, err, "failed to create temporary file")

			defer func() { _ = os.Remove(tmpFile.Name()) }()

			err = SetTags(tmpFile, tt.tags...)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr)

				return
			} else {
				assert.Empty(t, tt.wantErr)
			}

			got, err := GetTags(tmpFile)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr)

				return
			} else {
				assert.Empty(t, tt.wantErr)
			}

			assert.ElementsMatch(t, tt.tags, got)
		})
	}

}
