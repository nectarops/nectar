// SPDX-License-Identifier: AGPL-3.0-only

package nodeclient

import (
	"bytes"
	"testing"
)

func TestEmbeddedClientKeepsSwarmTokenOutOfDockerCLIArguments(t *testing.T) {
	t.Parallel()

	script := Script()
	if !bytes.HasPrefix(script, []byte("#!/usr/bin/env bash\n")) {
		t.Fatal("Script() does not contain the expected Bash client")
	}
	if bytes.Contains(script, []byte("docker swarm join --token")) {
		t.Fatal("Script() passes the Swarm token through process arguments")
	}
	if !bytes.Contains(script, []byte(`--data-binary "@${join_request_file}"`)) {
		t.Fatal("Script() does not submit the protected Swarm join request file")
	}
}

func TestEmbeddedClientConsumesDockerPackageVersionOutput(t *testing.T) {
	t.Parallel()

	script := Script()
	unsafeSelection := []byte(`$3 ~ ("^5:" requested "-") {print $3; exit}`)
	if bytes.Contains(script, unsafeSelection) {
		t.Fatal("Script() exits the Docker package-version pipeline before apt-cache finishes")
	}

	safeSelection := []byte(`$3 ~ ("^5:" requested "-") && !selected {print $3; selected = 1}`)
	if !bytes.Contains(script, safeSelection) {
		t.Fatal("Script() does not consume apt-cache output while selecting the first Docker package version")
	}
}
