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

package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterDocuments(t *testing.T) {
	// Sample documents
	docs := []Document{
		{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
		{EncodedName: "5678", Name: "Document2", Tags: []string{"tag2", "urgent"}},
		{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
		{EncodedName: "121314", Name: "DocArchive1", Tags: []string{"archive", "tag3"}},
	}

	testCases := []struct {
		name     string
		filter   string
		expected []Document
	}{
		{
			name:   "Exact Filter by Name",
			filter: "n == 'Document1'",
			expected: []Document{
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
			},
		},
		{
			name:   "Regex Filter by Name",
			filter: "n =~ '^Document[0-9]+$'",
			expected: []Document{
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
				{EncodedName: "5678", Name: "Document2", Tags: []string{"tag2", "urgent"}},
				{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
			},
		},
		{
			name:   "Regex Filter by Name literal",
			filter: "name =~ '^Document[0-9]+$'",
			expected: []Document{
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
				{EncodedName: "5678", Name: "Document2", Tags: []string{"tag2", "urgent"}},
				{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
			},
		},
		{
			name:   "Filter by Tag",
			filter: "t('urgent')",
			expected: []Document{
				{EncodedName: "5678", Name: "Document2", Tags: []string{"tag2", "urgent"}},
			},
		},
		{
			name:   "Combined Regex Filter and",
			filter: "n =~ '^Document3$' && t('archive')",
			expected: []Document{
				{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
			},
		},
		{
			name:   "Regex Match All Docs with 'archive' Tag",
			filter: "t('archive')",
			expected: []Document{
				{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
				{EncodedName: "121314", Name: "DocArchive1", Tags: []string{"archive", "tag3"}},
			},
		},
		{
			name:   "Regex Match All Docs with 'archive' Tag singleton first",
			filter: "t('archive') || n == 'Document1' && t('important')",
			expected: []Document{
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
				{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
				{EncodedName: "121314", Name: "DocArchive1", Tags: []string{"archive", "tag3"}},
			},
		},
		{
			name:   "Regex Match All Docs with 'archive' Tag singleton last",
			filter: "t('tag1') && n =~ 'Document' || t('important')",
			expected: []Document{
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
				{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
			},
		},
		{
			name:   "multiple ands",
			filter: "t('tag1') && n =~ 'Document' && t('important')",
			expected: []Document{
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
			},
		},
		{
			name:   "multiple ors",
			filter: "t('tag1') || n =~ 'Document' || t('important')",
			expected: []Document{
				{EncodedName: "5678", Name: "Document2", Tags: []string{"tag2", "urgent"}},
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
				{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
			},
		},
		{
			name:   "intersecting tags",
			filter: "t('tag2') && t('urgent')",
			expected: []Document{
				{EncodedName: "5678", Name: "Document2", Tags: []string{"tag2", "urgent"}},
			},
		},
		{
			name:   "intersecting and name",
			filter: "t('tag2') && n =~ 'Doc'",
			expected: []Document{
				{EncodedName: "5678", Name: "Document2", Tags: []string{"tag2", "urgent"}},
			},
		},
		{
			name:     "nothing matches",
			filter:   "t('tag2') && n =~ 'Doc' && t('important')",
			expected: []Document{},
		},
		{
			name:   "using not equal",
			filter: "n != 'Document1'",
			expected: []Document{
				{EncodedName: "5678", Name: "Document2", Tags: []string{"tag2", "urgent"}},
				{EncodedName: "91011", Name: "Document3", Tags: []string{"tag1", "archive"}},
				{EncodedName: "121314", Name: "DocArchive1", Tags: []string{"archive", "tag3"}},
			},
		},
		{
			name:   "filter by size",
			filter: "s >= 1",
			expected: []Document{
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
			},
		},
		{
			name:   "filter by inclusive tags",
			filter: "ti('tag1', 'important')",
			expected: []Document{
				{EncodedName: "1234", Name: "Document1", Tags: []string{"tag1", "important"}, Size: 1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := FilterDocuments(tc.filter, docs)
			require.NoError(t, err)

			if len(tc.expected) == 0 && len(result) == 0 {
				return
			}

			assert.ElementsMatch(t, tc.expected, result)

		})
	}
}
