package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestChromeProfileDirUsesSnapCommon(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("snap browser paths are Linux-specific")
	}

	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "cache"))

	got, err := chromeProfileDir("/snap/bin/chromium")
	if err != nil {
		t.Fatalf("chromeProfileDir() error = %v", err)
	}
	want := filepath.Join(home, "snap", "chromium", "common", "kubeql", "chrome")
	if got != want {
		t.Fatalf("chromeProfileDir() = %q, want %q", got, want)
	}
	assertPrivateDirectory(t, got)
}

func TestChromeProfileDirUsesUserCacheForRegularBrowser(t *testing.T) {
	cacheRoot := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LocalAppData", cacheRoot)
	case "darwin", "ios":
		t.Setenv("HOME", cacheRoot)
	case "plan9":
		t.Skip("test does not override plan9's user cache directory")
	default:
		t.Setenv("XDG_CACHE_HOME", cacheRoot)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("os.UserCacheDir() error = %v", err)
	}

	got, err := chromeProfileDir("/usr/bin/google-chrome")
	if err != nil {
		t.Fatalf("chromeProfileDir() error = %v", err)
	}
	want := filepath.Join(cache, "kubeql", "chrome")
	if got != want {
		t.Fatalf("chromeProfileDir() = %q, want %q", got, want)
	}
	assertPrivateDirectory(t, got)
}

func assertPrivateDirectory(t *testing.T, dir string) {
	t.Helper()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat profile directory: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("profile path %q is not a directory", dir)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("profile directory permissions = %o, want %o", got, want)
	}
}
