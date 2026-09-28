package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEntrypointRefreshesExtensions(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to test the container entrypoint")
	}
	t.Chdir(t.TempDir())
	if err := createEntrypoint(); err != nil {
		t.Fatal(err)
	}
	entrypoint, err := os.ReadFile("entrypoint.sh")
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("bash", "-n", "entrypoint.sh").CombinedOutput(); err != nil {
		t.Fatalf("entrypoint syntax: %v\n%s", err, output)
	}

	// Run the generated startup steps in an isolated tree, stopping before the
	// container-specific ownership changes and gosu invocation.
	script, _, ok := strings.Cut(string(entrypoint), "# Fix volume ownership")
	if !ok {
		t.Fatal("cannot locate privilege setup in entrypoint")
	}
	root := t.TempDir()
	script = strings.ReplaceAll(script, "/opt/", root+"/")
	bundled := filepath.Join(root, "yougile-bundled-extensions")
	volume := filepath.Join(root, "yougile", "extensions")
	write := func(path, content string, mode os.FileMode) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	check := func(path, expected string) {
		t.Helper()
		got, err := os.ReadFile(path)
		if err != nil || string(got) != expected {
			t.Fatalf("%s: got %q, error %v; want %q", path, got, err, expected)
		}
	}
	run := func() {
		t.Helper()
		if output, err := exec.Command("bash", "-c", script).CombinedOutput(); err != nil {
			t.Fatalf("startup: %v\n%s", err, output)
		}
	}

	write(filepath.Join(bundled, "builtin", "index.js"), "version 1", 0644)
	if err := os.MkdirAll(filepath.Dir(volume), 0755); err != nil {
		t.Fatal(err)
	}
	run()
	check(filepath.Join(volume, "builtin", "index.js"), "version 1")

	write(filepath.Join(volume, "custom", "index.js"), "custom extension", 0644)
	write(filepath.Join(volume, "builtin", "settings.json"), "custom settings", 0644)
	write(filepath.Join(bundled, "builtin", "index.js"), "version 2", 0644)
	write(filepath.Join(bundled, ".metadata"), "hidden file", 0644)
	write(filepath.Join(bundled, "run.sh"), "#!/bin/sh\nexit 0\n", 0755)
	for range 2 {
		run()
		check(filepath.Join(volume, "builtin", "index.js"), "version 2")
		check(filepath.Join(volume, "custom", "index.js"), "custom extension")
		check(filepath.Join(volume, "builtin", "settings.json"), "custom settings")
		check(filepath.Join(volume, ".metadata"), "hidden file")
	}
	info, err := os.Stat(filepath.Join(volume, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Fatal("extension script lost its executable permission")
	}

	// A missing source must stop startup instead of silently serving stale files.
	if err := os.Rename(bundled, bundled+"-missing"); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("bash", "-c", script+"\nprintf 'server started'\n").Run(); err == nil {
		t.Fatal("startup succeeded despite an extension copy failure")
	}
}
