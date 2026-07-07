package main

import (
	"fmt"
	"strings"
)

// CreateDockerFiles creates Dockerfile, entrypoint.sh, docker-compose.yml, and .dockerignore.
func CreateDockerFiles() error {
	dockerFiles := []struct {
		name     string
		function func() error
	}{
		{"Dockerfile", createDockerfile},
		{"entrypoint.sh", createEntrypoint},
		{"docker-compose.yml", createDockerCompose},
		{".dockerignore", createDockerIgnore},
	}

	for _, file := range dockerFiles {
		fmt.Printf("  %s...\n", file.name)
		if err := file.function(); err != nil {
			return fmt.Errorf(tr("ошибка создания %s: %v", "error creating %s: %v"), file.name, err)
		}
	}

	return nil
}

func createDockerfile() error {
	if !regenConfigs && fileExists("Dockerfile") {
		fmt.Println("    [SKIP] " + tr("Файл уже существует, пропускаем", "File already exists, skipping"))
		return nil
	}
	content := fmt.Sprintf(`# YouGile Docker Image
FROM %s

%s
RUN apt-get update \
    && apt-get install -y --no-install-recommends gosu \
    && rm -rf /var/lib/apt/lists/* \
    && gosu nobody true

%s
ADD --chown=node:node yougile.tar.gz /opt/

WORKDIR /opt/yougile

%s
RUN mkdir -p database logs user-data extensions \
    && chown node:node database logs user-data extensions

EXPOSE 8001

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["./server"]`,
		nodeImage(),
		"# Install gosu for privilege dropping in entrypoint",
		"# Extract the archive with node ownership. Dependencies (node_modules)\n# are vendored inside the archive — no npm install is needed.",
		"# Create volume mount points owned by the node user",
	)
	return writeFile("Dockerfile", content)
}

func createEntrypoint() error {
	if !regenConfigs && fileExists("entrypoint.sh") {
		fmt.Println("    [SKIP] " + tr("Файл уже существует, пропускаем", "File already exists, skipping"))
		return nil
	}
	content := `#!/bin/bash
set -e

# Set up config symlinks
rm -f /opt/yougile/conf.json /opt/yougile/license.key
ln -sf /opt/yougile-config/conf.json /opt/yougile/conf.json
ln -sf /opt/yougile-config/license.key /opt/yougile/license.key

# Ensure required subdirs exist in volumes
mkdir -p /opt/yougile/database/companies
mkdir -p /opt/yougile/database/user-events

# Fix volume ownership before dropping privileges. Only touches files
# not already owned by node, so restarts don't rewrite the whole tree.
find /opt/yougile/database /opt/yougile/user-data \
     /opt/yougile/logs /opt/yougile/extensions \
  ! -user node -exec chown node:node {} + || true

# Drop to non-root and exec
exec gosu node "$@"
`
	return writeFile("entrypoint.sh", content)
}

func createDockerCompose() error {
	if !regenConfigs && fileExists("docker-compose.yml") {
		fmt.Println("    [SKIP] " + tr("Файл уже существует, пропускаем", "File already exists, skipping"))
		return nil
	}

	var b strings.Builder
	b.WriteString("services:\n")
	b.WriteString(yougileServiceSection())
	if nginxEnabled() {
		b.WriteString(nginxServiceSection())
		b.WriteString(certbotServiceSection())
	}
	b.WriteString("\n")
	b.WriteString("# Named volumes for YouGile data\n")
	b.WriteString("volumes:\n")
	b.WriteString("  yougile_database:\n")
	b.WriteString("  yougile_userdata:\n")
	b.WriteString("  yougile_logs:\n")
	b.WriteString("  yougile_extensions:\n")

	return writeFile("docker-compose.yml", b.String())
}

func yougileServiceSection() string {
	var b strings.Builder
	b.WriteString("  yougile:\n")
	b.WriteString("    build:\n")
	b.WriteString("      context: .\n")
	b.WriteString("      dockerfile: Dockerfile\n")
	b.WriteString("    container_name: yougile\n")
	b.WriteString("    hostname: yougile\n")
	b.WriteString("    ports:\n")
	// When nginx is the reverse proxy, bind 8001 to localhost only to prevent
	// clients from bypassing nginx and hitting the backend directly.
	if nginxEnabled() {
		b.WriteString("      - \"127.0.0.1:8001:8001\"\n")
	} else {
		b.WriteString("      - \"8001:8001\"\n")
	}
	b.WriteString("    healthcheck:\n")
	b.WriteString("      test: [\"CMD\", \"wget\", \"--no-verbose\", \"--tries=1\", \"--spider\", \"http://localhost:8001\"]\n")
	b.WriteString("      interval: 60s\n")
	b.WriteString("      retries: 5\n")
	b.WriteString("      start_period: 20s\n")
	b.WriteString("      timeout: 10s\n")
	b.WriteString("    volumes:\n")
	b.WriteString("      # Named volumes for persistent data\n")
	b.WriteString("      - yougile_database:/opt/yougile/database\n")
	b.WriteString("      - yougile_userdata:/opt/yougile/user-data\n")
	b.WriteString("      - yougile_logs:/opt/yougile/logs\n")
	b.WriteString("      - yougile_extensions:/opt/yougile/extensions\n")
	b.WriteString("      # Bind mount for config files (conf.json, license.key)\n")
	b.WriteString("      - ./yougile:/opt/yougile-config:rw\n")
	b.WriteString("    environment:\n")
	b.WriteString("      - NODE_ENV=production\n")
	b.WriteString("      - CHOKIDAR_USEPOLLING=true\n")
	b.WriteString("      - CHOKIDAR_INTERVAL=1000\n")
	b.WriteString("    cap_drop:\n")
	b.WriteString("      - ALL\n")
	b.WriteString("    cap_add:\n")
	b.WriteString("      - CHOWN\n")
	// DAC_OVERRIDE: the image tree is owned by node (ADD --chown), so the
	// root entrypoint needs it to manage files in node-owned directories.
	b.WriteString("      - DAC_OVERRIDE\n")
	b.WriteString("      - SETUID\n")
	b.WriteString("      - SETGID\n")
	b.WriteString("    security_opt:\n")
	b.WriteString("      - no-new-privileges:true\n")
	b.WriteString(resourceLimitsSection(yougileMemLimit(), yougileCPULimit()))
	b.WriteString("    restart: unless-stopped\n")
	b.WriteString("\n")
	return b.String()
}

func nginxServiceSection() string {
	var b strings.Builder
	b.WriteString("  nginx:\n")
	b.WriteString("    image: " + nginxImage() + "\n")
	b.WriteString("    ports:\n")
	b.WriteString("      - \"" + nginxHTTPPort() + ":80\"\n")
	b.WriteString("      - \"" + nginxHTTPSPort() + ":443\"\n")
	b.WriteString("    restart: unless-stopped\n")
	b.WriteString("    volumes:\n")
	b.WriteString("      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro\n")
	b.WriteString("      - ./nginx/conf.d:/etc/nginx/conf.d:ro\n")
	b.WriteString("      - ./certbot/www:/var/www/certbot:ro\n")
	b.WriteString("      - ./certbot/conf:/etc/nginx/ssl:ro\n")
	b.WriteString("      - ./nginx/logs:/var/log/nginx:rw\n")
	b.WriteString("      # Read-only user-data for direct file serving (see the\n")
	b.WriteString("      # /user-data/ block in nginx.conf)\n")
	b.WriteString("      - yougile_userdata:/opt/yougile/user-data:ro\n")
	b.WriteString("    cap_drop:\n")
	b.WriteString("      - ALL\n")
	b.WriteString("    cap_add:\n")
	b.WriteString("      - NET_BIND_SERVICE\n")
	b.WriteString("    security_opt:\n")
	b.WriteString("      - no-new-privileges:true\n")
	b.WriteString(resourceLimitsSection(nginxMemLimit(), nginxCPULimit()))
	b.WriteString("    depends_on:\n")
	b.WriteString("      yougile:\n")
	b.WriteString("        condition: service_healthy\n")
	b.WriteString("\n")
	return b.String()
}

func certbotServiceSection() string {
	var b strings.Builder
	b.WriteString("  # Certbot — certificate management only (run via: docker compose run --rm certbot)\n")
	b.WriteString("  certbot:\n")
	b.WriteString("    image: " + certbotImage() + "\n")
	b.WriteString("    volumes:\n")
	b.WriteString("      - ./certbot/www:/var/www/certbot:rw\n")
	b.WriteString("      - ./certbot/conf:/etc/letsencrypt:rw\n")
	b.WriteString("    profiles: [\"tools\"]\n")
	b.WriteString("\n")
	return b.String()
}

// resourceLimitsSection returns the deploy.resources.limits YAML block,
// or an empty string if neither limit is configured.
func resourceLimitsSection(memLimit, cpuLimit string) string {
	if memLimit == "" && cpuLimit == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("    deploy:\n")
	b.WriteString("      resources:\n")
	b.WriteString("        limits:\n")
	if memLimit != "" {
		b.WriteString("          memory: " + memLimit + "\n")
	}
	if cpuLimit != "" {
		b.WriteString("          cpus: '" + cpuLimit + "'\n")
	}
	return b.String()
}

func createDockerIgnore() error {
	if !regenConfigs && fileExists(".dockerignore") {
		fmt.Println("    [SKIP] " + tr("Файл уже существует, пропускаем", "File already exists, skipping"))
		return nil
	}
	// Whitelist approach: the image build only needs the archive and the
	// entrypoint, so exclude everything else (including .env with secrets
	// and the installer binary).
	content := `# Exclude everything from the build context except what the image needs
*
!yougile.tar.gz
!entrypoint.sh`
	return writeFile(".dockerignore", content)
}
