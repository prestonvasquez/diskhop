//go:build mage
// +build mage

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AddLicense adds the Apache License header to all .go files in the current directory and subdirectories
func AddLicense() error {
	const licenseText = `// Copyright 2024 Preston Vasquez
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
`

	fmt.Println("Adding license to all .go files...")

	// Walk through all .go files in the current directory and subdirectories
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Process only .go files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			// Read the contents of the file
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %v", path, err)
			}

			// Check if the license is already present
			if strings.Contains(string(data), "Licensed under the Apache License, Version 2.0") {
				fmt.Printf("License already exists in %s\n", path)
				return nil
			}

			err = os.WriteFile(path, []byte(licenseText+"\n"+string(data)), 0o644)
			if err != nil {
				return fmt.Errorf("failed to write license to file %s: %v", path, err)
			}

			fmt.Printf("Added license to %s\n", path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to add license to files: %v", err)
	}

	fmt.Println("License added successfully to all .go files.")
	return nil
}

// Build compiles the project and moves the binary to the GOPATH/bin directory.
func Build() error {
	// Change directory to cmd before building
	if err := os.Chdir("cmd"); err != nil {
		return fmt.Errorf("Failed to change directory to cmd: %v", err)
	}

	// Build the Go binary with specific tags and output it as "diskhop-beta"
	cmd := exec.Command("go", "build", "-tags=cse", "-o", "diskhop-beta", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to build binary: %v", err)
	}

	// Move the binary to the GOPATH/bin directory and rename it to "dop"
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return errors.New("GOPATH is not set")
	}

	binPath := filepath.Join(gopath, "bin", "dop")
	if err := os.Rename("diskhop-beta", binPath); err != nil {
		return fmt.Errorf("Failed to move binary to %s: %v", binPath, err)
	}

	return nil
}

// Fmt runs `go fmt` on all Go files in the project
func Fmt() error {
	fmt.Println("Formatting all Go files...")

	// Run `go fmt ./...` to format all Go files in the project
	cmd := exec.Command("gofumpt", "-l", "-w", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("go fmt failed: %v", err)
	}

	fmt.Println("All Go files formatted successfully.")
	return nil
}

// Lint runs golangci-lint with the current configuration
func Lint() error {
	fmt.Println("skipping lint")
	return nil

	fmt.Println("Running golangci-lint...")

	// Check if golangci-lint is installed
	_, err := exec.LookPath("golangci-lint")
	if err != nil {
		return fmt.Errorf("golangci-lint not found: %v", err)
	}

	// Run golangci-lint
	cmd := exec.Command("golangci-lint", "run")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("golangci-lint failed: %v", err)
	}

	fmt.Println("golangci-lint completed successfully.")

	return nil
}

// Test runs all tests in the project.
func Test() error {
	fmt.Println("Running all tests...")

	// Run `go test ./...` to execute all tests in the project
	cmd := exec.Command("go", "test", "-v", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("tests failed: %v", err)
	}

	fmt.Println("All tests passed successfully.")
	return nil
}

// TestMongo runs the integration tests for the MongoDB store locally.
func TestMongo() error {
	// Change to the directory where the test file is located
	if err := os.Chdir("store/mongodop/test"); err != nil {
		return fmt.Errorf("failed to change directory: %v", err)
	}

	cmd := exec.Command("go", "test", "-v", "-failfast")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run integration tests: %v", err)
	}

	return nil
}

// mageLogger implements io.Writer to log output from exec.Command
type mageLogger struct{}

func (l mageLogger) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return len(p), nil
}

// TestOsUtilUbuntu2204 runs the os-specific unit tests in a Docker container
// using the Ubuntu 22.04 image.
func TestOsUtilUbuntu2204() error {
	const imageName = "os-ubuntu2204"
	const dockerfilePath = "docker/ubuntu2204.dockerfile"

	// Step 1: Build the Docker image
	fmt.Println("Building Docker image:", imageName)
	buildCmd := exec.Command("docker", "build", "--platform=linux/amd64", "-t", imageName, "-f", dockerfilePath, ".")
	buildCmd.Stdout = mageLogger{}
	buildCmd.Stderr = mageLogger{}

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	fmt.Println("Docker image built successfully:", imageName)

	// Step 2: Determine the test command to run (default to mongodop package)
	const testCommand = "cd internal/osutil/ && go test -v ./..."

	// Step 3: Run the Docker container with the specified test command
	fmt.Println("Running test command in Docker container:", testCommand)
	runCmd := exec.Command("docker", "run", "--rm", imageName, testCommand)
	runCmd.Stdout = mageLogger{}
	runCmd.Stderr = mageLogger{}

	if err := runCmd.Run(); err != nil {
		return fmt.Errorf("failed to run test command in Docker container: %w", err)
	}

	fmt.Println("Test command executed successfully.")
	return nil
}
