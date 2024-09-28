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

import "context"

// Reverter is an interface that defines the behavior of reverting.
type Reverter interface {
	// Revert will DELETE the files associated with the SHA in ALL cases.
	// This is a WIP and will be updated to support more complex behavior.
	//
	// Deprecatd: DO NOT USE IN PRODUCTION, SEE DESCRIPTION.
	Revert(ctx context.Context, sha string) error
}
