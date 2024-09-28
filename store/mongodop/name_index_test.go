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

//func TestUnionNames(t *testing.T) {
//	tests := []struct {
//		name    string
//		nidx    nameIndex
//		names   []string
//		want    []string
//		wantErr bool
//	}{
//		{
//			name: "single match",
//			nidx: nameIndex{nameToFile: map[string]gridfs.File{
//				"file1.txt": gridfs.File{Name: "File One"},
//				"file2.txt": gridfs.File{Name: "File Two"},
//			}},
//			names:   []string{"file1.*"},
//			want:    []string{"File One"},
//			wantErr: false,
//		},
//		{
//			name: "multiple matches",
//			nidx: nameIndex{nameToFile: map[string]gridfs.File{
//				"file1.txt": gridfs.File{Name: "File One"},
//				"file2.txt": gridfs.File{Name: "File Two"},
//				"note.txt":  gridfs.File{Name: "Note File"},
//			}},
//			names:   []string{"file.*", "note.*"},
//			want:    []string{"File One", "File Two", "Note File"},
//			wantErr: false,
//		},
//		{
//			name: "no match",
//			nidx: nameIndex{nameToFile: map[string]gridfs.File{
//				"file1.txt": gridfs.File{Name: "File One"},
//				"file2.txt": gridfs.File{Name: "File Two"},
//			}},
//			names:   []string{"nonexistent.*"},
//			want:    []string{},
//			wantErr: false,
//		},
//		{
//			name: "invalid regex",
//			nidx: nameIndex{nameToFile: map[string]gridfs.File{
//				"file1.txt": gridfs.File{Name: "File One"},
//			}},
//			names:   []string{"[\\"}, // Backslash should trigger an error in regex compilation
//			want:    nil,
//			wantErr: true,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := unionNames(tt.nidx, tt.names...)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("unionNames() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			assert.Equal(t, tt.want, got)
//		})
//	}
//}
//
//func TestIntersectNames(t *testing.T) {
//	tests := []struct {
//		name    string
//		nidx    nameIndex
//		names   []string
//		want    []string
//		wantErr bool
//	}{
//		{
//			name: "all match",
//			nidx: nameIndex{nameToFile: map[string]gridfs.File{
//				"file1.txt": gridfs.File{Name: "File One"},
//				"file2.txt": gridfs.File{Name: "File Two"},
//			}},
//			names:   []string{"file.*", ".*1.*"},
//			want:    []string{"File One"},
//			wantErr: false,
//		},
//		{
//			name: "partial match",
//			nidx: nameIndex{nameToFile: map[string]gridfs.File{
//				"file1.txt": gridfs.File{Name: "File One"},
//				"file2.txt": gridfs.File{Name: "File Two"},
//			}},
//			names:   []string{"file.*", ".*3.*"},
//			want:    []string{},
//			wantErr: false,
//		},
//		{
//			name: "no match due to invalid regex",
//			nidx: nameIndex{nameToFile: map[string]gridfs.File{
//				"file1.txt": gridfs.File{Name: "File One"},
//			}},
//			names:   []string{"[\\"}, // Invalid regex
//			want:    nil,
//			wantErr: true,
//		},
//		{
//			name: "no regex provided",
//			nidx: nameIndex{nameToFile: map[string]gridfs.File{
//				"file1.txt": gridfs.File{Name: "File One"},
//			}},
//			names:   []string{},
//			want:    nil,
//			wantErr: true,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := intersectNames(tt.nidx, tt.names...)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("intersectNames() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			assert.Equal(t, tt.want, got)
//		})
//	}
//}
