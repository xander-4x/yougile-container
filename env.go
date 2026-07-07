package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"strings"
)

// containerRuntime returns the container runtime command: "docker" by default,
// or the value of CONTAINER_RUNTIME (e.g. "podman").
func containerRuntime() string {
	if r := os.Getenv("CONTAINER_RUNTIME"); r != "" {
		return r
	}
	return "docker"
}

func nginxEnabled() bool {
	v := os.Getenv("NGINX_ENABLED")
	return v == "" || strings.ToLower(v) == "true" || v == "1"
}

func yougileMemLimit() string { return os.Getenv("YOUGILE_MEM_LIMIT") }
func yougileCPULimit() string { return os.Getenv("YOUGILE_CPU_LIMIT") }
func nginxMemLimit() string   { return os.Getenv("NGINX_MEM_LIMIT") }
func nginxCPULimit() string   { return os.Getenv("NGINX_CPU_LIMIT") }

//go:embed .env.example
var envExample []byte

// writeEnvExample writes the embedded .env.example to disk if it does not exist yet.
func writeEnvExample() {
	if _, err := os.Stat(".env.example"); err == nil {
		return
	}
	if err := os.WriteFile(".env.example", envExample, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write .env.example: %v\n", err)
	}
}

// loadDotEnv reads .env and populates environment variables not already set.
func loadDotEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// strip surrounding quotes
		if len(val) >= 2 &&
			((val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		// shell env takes precedence over .env file
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}

func youGileDownloadURL() string {
	if u := os.Getenv("YOUGILE_DOWNLOAD_URL"); u != "" {
		return u
	}
	return "https://dist.yougile.com/linux/latest/yougile.tar.gz"
}

func nodeImage() string {
	if img := os.Getenv("YOUGILE_NODE_IMAGE"); img != "" {
		return img
	}
	return "node:22.3.0"
}

func nginxImage() string {
	if img := os.Getenv("YOUGILE_NGINX_IMAGE"); img != "" {
		return img
	}
	return "nginx:alpine"
}

func certbotImage() string {
	if img := os.Getenv("YOUGILE_CERTBOT_IMAGE"); img != "" {
		return img
	}
	return "certbot/certbot:latest"
}

func nginxHTTPPort() string {
	if p := os.Getenv("NGINX_HTTP_PORT"); p != "" {
		return p
	}
	return "80"
}

func nginxHTTPSPort() string {
	if p := os.Getenv("NGINX_HTTPS_PORT"); p != "" {
		return p
	}
	return "443"
}
