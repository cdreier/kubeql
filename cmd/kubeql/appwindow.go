package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func findChrome() (string, error) {
	if p := os.Getenv("CHROME_PATH"); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
		return "", fmt.Errorf("CHROME_PATH %q is not a file", p)
	}

	names := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"brave-browser",
		"microsoft-edge",
		"microsoft-edge-stable",
	}
	for _, name := range names {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}

	if runtime.GOOS == "darwin" {
		mac := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if st, err := os.Stat(mac); err == nil && !st.IsDir() {
			return mac, nil
		}
	}

	return "", fmt.Errorf("no Chrome/Chromium on PATH (install one or set CHROME_PATH)")
}

func chromeProfileDir(bin string) (string, error) {
	base, err := chromeProfileBaseDir(bin)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "kubeql", "chrome")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("chrome profile: %w", err)
	}
	return dir, nil
}

func chromeProfileBaseDir(bin string) (string, error) {
	// Strictly confined snaps cannot write to the real user's hidden ~/.cache
	// directory. Keep their profile in the snap's revision-independent user data
	// directory instead so localStorage survives snap refreshes.
	if runtime.GOOS == "linux" {
		cleanBin := filepath.Clean(bin)
		if filepath.Dir(cleanBin) == "/snap/bin" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("snap chrome profile: user home directory unavailable: %w", err)
			}
			if home == "" {
				return "", fmt.Errorf("snap chrome profile: user home directory unavailable")
			}
			return filepath.Join(home, "snap", filepath.Base(cleanBin), "common"), nil
		}
	}

	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return base, nil
}

// startAppWindow launches Chrome in --app mode with a persistent user-data-dir
// so we get a dedicated process we can wait on (existing Chrome would otherwise
// take the window and exit our child immediately) and localStorage survives.
func startAppWindow(url string) (*exec.Cmd, error) {
	bin, err := findChrome()
	if err != nil {
		return nil, err
	}
	profile, err := chromeProfileDir(bin)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin,
		"--app="+url,
		"--user-data-dir="+profile,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--disable-sync",
		"--disable-translate",
		"--disable-features=Translate,TranslateUI",
		"--window-size=1600,840",
		"--class=kubeql",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start chrome: %w", err)
	}
	return cmd, nil
}
