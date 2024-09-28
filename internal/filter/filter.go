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

import "github.com/Knetic/govaluate"

type Document struct {
	EncodedName string
	Name        string
	Tags        []string
	Size        int64
}

func FilterDocuments(expression string, documents []Document) ([]Document, error) {
	var filteredDocs []Document
	for _, doc := range documents {
		// Evaluate the expression against the document
		match, err := evaluateExpression(expression, doc)
		if err != nil {
			return nil, err
		}

		// If the document matches the expression, add it to the filtered list
		if match {
			filteredDocs = append(filteredDocs, doc)
		}
	}

	return filteredDocs, nil
}

func (doc Document) HasAllTags(args ...interface{}) (interface{}, error) {
	tagSet := make(map[string]bool)
	for _, tag := range doc.Tags {
		tagSet[tag] = true
	}

	for _, arg := range args {
		if !tagSet[arg.(string)] {
			return false, nil
		}
	}

	return true, nil
}

func (doc Document) HasNoTags(args ...interface{}) (interface{}, error) {
	tagSet := make(map[string]bool)
	for _, tag := range doc.Tags {
		tagSet[tag] = true
	}

	for _, arg := range args {
		if tagSet[arg.(string)] {
			return false, nil
		}
	}

	return true, nil
}

func (doc Document) HasTag(args ...interface{}) (interface{}, error) {
	tagSet := make(map[string]bool)
	for _, tag := range doc.Tags {
		tagSet[tag] = true
	}

	for _, arg := range args {
		if tagSet[arg.(string)] {
			return true, nil
		}
	}

	return false, nil
}

// evaluateExpression takes a string expression and evaluates it against the document
func evaluateExpression(expString string, doc Document) (bool, error) {
	if expString == "" {
		return true, nil
	}

	// Map to hold the document's fields for evaluation
	parameters := make(map[string]interface{})

	parameters["name"] = doc.Name
	parameters["size"] = doc.Size

	parameters["n"] = doc.Name
	parameters["s"] = doc.Size

	// Custom function to check if the document has the specified tag
	functions := map[string]govaluate.ExpressionFunction{
		"tag":          doc.HasTag,
		"t":            doc.HasTag,
		"tagInclusive": doc.HasAllTags,
		"ti":           doc.HasAllTags,
		"noTag":        doc.HasNoTags,
		"nt":           doc.HasNoTags,
	}

	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expString, functions)
	if err != nil {
		return false, err
	}

	// Evaluate the expression against the document
	result, err := expression.Evaluate(parameters)
	if err != nil {
		return false, err
	}

	// Convert the result to a boolean value
	return result.(bool), nil
}
