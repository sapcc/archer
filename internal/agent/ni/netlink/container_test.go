// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package netlink

import (
	"context"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/sapcc/go-bits/osext"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestNetlinkPrivilegedContainer runs all Linux netlink tests inside a
// privileged container. This test can be run from any OS (macOS, Linux, etc.)
// as long as a container runtime (Podman/Docker) is available.
func TestNetlinkPrivilegedContainer(t *testing.T) {
	// Skip in CI environments where container runtime is not available
	if osext.GetenvBool("CHECK_SKIPS_FUNCTIONAL_TEST") {
		t.Skip("Skipping functional test as CHECK_SKIPS_FUNCTIONAL_TEST is set")
	}

	// If we're already inside the privileged container, skip this orchestrator test
	// (the actual tests in netlink_privileged_test.go will run directly)
	if osext.GetenvBool("NETLINK_TEST_PRIVILEGED") {
		t.Skip("Already inside privileged container, skipping orchestrator")
	}

	ctx := context.Background()

	// Get root path of the project
	_, b, _, _ := runtime.Caller(0)
	rootpath := filepath.Join(filepath.Dir(b), "../../../..")

	req := testcontainers.ContainerRequest{
		Image: "golang:1.26-alpine",
		Cmd:   []string{"sleep", "infinity"},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Privileged = true
			hc.Binds = []string{rootpath + ":/src"}
		},
		ConfigModifier: func(cc *container.Config) {
			cc.WorkingDir = "/src"
		},
		Env: map[string]string{
			"NETLINK_TEST_PRIVILEGED": "true",
			"CGO_ENABLED":             "0",
		},
		WaitingFor: wait.ForExec([]string{"true"}),
	}

	ctr, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
			ProviderType:     testcontainers.ProviderPodman,
		})
	require.NoError(t, err, "Failed to start privileged container")
	defer func() { _ = ctr.Terminate(ctx) }()

	// Run the Linux-specific tests inside the container
	exitCode, outputReader, err := ctr.Exec(ctx, []string{
		"go", "test", "-v",
		"-run", "TestNetlinkSuite",
		"./internal/agent/ni/netlink/...",
	})
	require.NoError(t, err, "Failed to exec tests in container")

	// Read output
	outputBytes, err := io.ReadAll(outputReader)
	require.NoError(t, err, "Failed to read test output")
	output := string(outputBytes)

	// Log output for debugging
	t.Log(output)

	// Check if tests passed
	if exitCode != 0 {
		// Extract just the failure summary if possible
		lines := strings.Split(output, "\n")
		var failLines []string
		for _, line := range lines {
			if strings.Contains(line, "FAIL") || strings.Contains(line, "Error") {
				failLines = append(failLines, line)
			}
		}
		if len(failLines) > 0 {
			t.Fatalf("Tests failed inside container (exit code %d):\n%s", exitCode, strings.Join(failLines, "\n"))
		}
		t.Fatalf("Tests failed inside container (exit code %d)", exitCode)
	}
}
