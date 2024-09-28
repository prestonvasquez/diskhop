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

package main

import "errors"

// errNotDiskhop is an error that indicates the directory is not a diskhop
// repository.
var errNotDiskhop = errors.New("not a diskhop repository")

// errConnStringEmpty represents an error where the connection string is
// empty.
var errConnStringEmpty = errors.New("connection string cannot be empty")
