package service

import (
	"sync"
	"testing"
)

func TestNormalizeProxyURLKeepsExplicitProxyOutsideContainer(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://172.17.0.1:17890")
	originalOnce := containerEnvOnce
	originalFlag := containerEnvFlag
	t.Cleanup(func() {
		containerEnvOnce = originalOnce
		containerEnvFlag = originalFlag
	})
	containerEnvOnce = sync.Once{}
	containerEnvFlag = false
	got := normalizeProxyURL("http://127.0.0.1:7890")
	if got != "http://127.0.0.1:7890" {
		t.Fatalf("expected explicit proxy to stay unchanged, got %q", got)
	}
}

func TestNormalizeProxyURLUsesContainerReachableProxyInsideContainer(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://172.17.0.1:17890")
	originalOnce := containerEnvOnce
	originalFlag := containerEnvFlag
	t.Cleanup(func() {
		containerEnvOnce = originalOnce
		containerEnvFlag = originalFlag
	})
	containerEnvOnce = sync.Once{}
	containerEnvFlag = true
	got := normalizeProxyURL("http://127.0.0.1:7890")
	if got != "http://172.17.0.1:17890" {
		t.Fatalf("expected env proxy in container, got %q", got)
	}
}

func TestNormalizeProxyURLKeepsNonLoopbackProxyInsideContainer(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://172.17.0.1:17890")
	originalOnce := containerEnvOnce
	originalFlag := containerEnvFlag
	t.Cleanup(func() {
		containerEnvOnce = originalOnce
		containerEnvFlag = originalFlag
	})
	containerEnvOnce = sync.Once{}
	containerEnvFlag = true
	got := normalizeProxyURL("http://proxy.example.com:8080")
	if got != "http://proxy.example.com:8080" {
		t.Fatalf("expected non-loopback proxy to stay unchanged, got %q", got)
	}
}
