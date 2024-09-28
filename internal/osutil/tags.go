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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pkg/xattr"
	"howett.net/plist"
)

var ErrFileNotExists = fmt.Errorf("file does not exist")

const darwinAttrListTag = "com.apple.metadata:_kMDItemUserTags"

// GetTags returns a list of file tags for the current operating system.
func GetTags(file *os.File) ([]string, error) {
	if file == nil {
		return nil, ErrFileNotExists
	}

	switch runtime.GOOS {
	case "darwin":
		return getDarwinTags(file.Name())
	case "linux":
		return getLinuxTags(file.Name())
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// SetTags sets a list of tags for a file on the current operating system.
func SetTags(file *os.File, tags ...string) error {
	if file == nil {
		return ErrFileNotExists
	}

	switch runtime.GOOS {
	case "darwin":
		return setDarwinTags(file.Name(), tags...)
	case "linux":
		return setLinuxTags(file.Name(), tags...)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func reindexSpotlight(directory string) error {
	cmd := exec.Command("mdutil", "-E", directory)
	err := cmd.Run()

	return err
}

// getDarwinTags retrieves tags from a file on macOS.
func getDarwinTags(filePath string) ([]string, error) {
	if err := reindexSpotlight(filePath); err != nil {
		return nil, err
	}

	// Retrieve xattr data
	list, err := xattr.Get(filePath, darwinAttrListTag)
	if err != nil {
		return nil, nil
	}

	// Unmarshal plist data into a slice of strings
	var colList []string
	_, err = plist.Unmarshal(list, &colList)
	if err != nil {
		return nil, err
	}

	toReturn := make([]string, len(colList), len(colList))

	for i, col := range colList {
		fmt.Sscanf(col, "%s", &toReturn[i])
	}

	return toReturn, nil
}

// setDarwinTags sets tags for a file on macOS.
func setDarwinTags(filePath string, tags ...string) error {
	var plistArrayElements string
	for _, tag := range tags {
		plistArrayElements += fmt.Sprintf("<string>%s</string>", tag)
	}

	plistArray := fmt.Sprintf("<array>%s</array>", plistArrayElements)
	plist := fmt.Sprintf(`<plist version="1.0">%s</plist>`, plistArray)

	docHeader := `<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`

	// Generate the PLIST content with static and dynamic parts
	plistContent := fmt.Sprintf("%s%s", docHeader, plist)

	// Use xattr to set the attribute from the generated PLIST content
	cmd := exec.Command("xattr", "-w", "com.apple.metadata:_kMDItemUserTags", plistContent, filePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// hasLinuxTags checks if the file has the 'user.tags' extended attribute.
func hasLinuxTags(filePath string) (bool, error) {
	// Use `getfattr` to list all extended attributes
	cmd := exec.Command("getfattr", "-d", filePath)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("error checking extended attributes: %v, stderr: %s", err, stderr.String())
	}

	// Check if the output contains the 'user.tags' attribute
	return strings.Contains(out.String(), "user.tags"), nil
}

// getLinuxTags retrieves tags from a file on Linux using extended attributes.
func getLinuxTags(filePath string) ([]string, error) {
	// First, check if the file has the 'user.tags' attribute
	hasTags, err := hasLinuxTags(filePath)
	if err != nil {
		return nil, err
	}
	if !hasTags {
		// If the file doesn't have the 'user.tags' attribute, return nil
		return nil, nil
	}

	// Use `getfattr` to retrieve the extended attribute with the tags
	cmd := exec.Command("getfattr", "-n", "user.tags", "--only-values", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	if out.String() == "" {
		return nil, nil
	}

	// Split the retrieved tag string into individual tags
	tags := strings.Split(strings.TrimSpace(out.String()), ",")

	return tags, nil
}

// setLinuxTags sets tags for a file on Linux using extended attributes.
func setLinuxTags(filePath string, tags ...string) error {
	// Join tags into a single string, separated by commas
	tagString := strings.Join(tags, ",")

	// Use `setfattr` to set the extended attribute with the tags
	cmd := exec.Command("setfattr", "-n", "user.tags", "-v", tagString, filePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
