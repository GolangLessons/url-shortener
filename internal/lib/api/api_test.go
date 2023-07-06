package api_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"url-shortener/internal/lib/api"
)

func TestGetRedirect_FileDescriptorsLeak(t *testing.T) {
	url := "http://example.com/"

	// to open file descriptors to the necessary network libraries
	_, _ = api.GetRedirect(url)

	beforeCall := countOpenFileDescriptors(t)
	_, _ = api.GetRedirect(url)
	afterCall := countOpenFileDescriptors(t)

	assert.Equal(t, beforeCall, afterCall)
}

func countOpenFileDescriptors(t *testing.T) int {
	command := fmt.Sprintf("lsof -p %v", os.Getpid())
	output, err := exec.Command("/bin/sh", "-c", command).Output()
	if err != nil {
		t.Fatalf("failed to run command '%s': %s", command, err)
	}
	lines := strings.Split(string(output), "\n")
	return len(lines) - 1
}
