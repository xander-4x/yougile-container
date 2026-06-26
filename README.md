# YouGile Container

Automated setup and deployment tool for YouGile project management, built on Docker and Docker Compose. Creates a fully containerized environment with an Nginx reverse proxy and SSL support.

## Requirements

**Required components:**

- Docker 20.10 or later
- Docker Compose 2.0 or later

**Server resources:**

- 2 GB RAM minimum (4 GB recommended)
- 5 GB free disk space minimum

**Supported operating systems:**

- Linux (Ubuntu, Debian, CentOS, RHEL, Fedora, and others)
- macOS
- Windows (with WSL2)

---

## Podman Support

Set `CONTAINER_RUNTIME=podman` in `.env` or in your shell to use Podman instead of Docker — the installer will invoke `podman compose` and `podman exec` accordingly:

```env
CONTAINER_RUNTIME=podman
```

**Rootless mode — privileged ports.** In rootless Podman, binding ports 80 and 443 requires a one-time kernel setting:

```bash
sudo sysctl -w net.ipv4.ip_unprivileged_port_start=80
```

Or persist it in `/etc/sysctl.d/99-unprivileged-ports.conf`.

**Rootless mode — capabilities and `gosu`.** `cap_drop: ALL` / `cap_add` and `gosu` rely on Linux capabilities that are not available inside a rootless user namespace. The deployment will still function, but those hardening settings have no effect. Run rootful Podman (`sudo podman`) to retain the full security profile.

**`deploy.resources.limits`.** Resource limit support requires `podman-compose` 1.0+ or Podman 4.4+ with the built-in compose subcommand. Older versions silently ignore the `deploy` block.

**Manual commands.** All `docker compose` / `docker stats` / `docker run` commands in this README should be replaced with their `podman` equivalents when running under Podman.

---

## Quick Start

### Step 1: Prepare the files

Two files are included in the delivery package. Place them in the same working directory and make the binary executable:

```
yougile-container      # installer
.env.example           # configuration template
```

```bash
chmod +x yougile-container
```

### Step 2: (Optional) Configure the environment

Copy `.env.example` to `.env` and edit the relevant settings **before** running the installation:

```bash
cp .env.example .env
# edit .env to match your environment
```

See [Configuration via .env](#configuration-via-env) for a full description of available options.

### Step 3: Run the installer

```bash
./yougile-container install
```

The installer automatically:

1. Creates the directory structure
2. Generates configuration files (`conf.json`, `nginx.conf`, `license.key`)
3. Creates Docker files (`Dockerfile`, `docker-compose.yml`, `.dockerignore`)
4. Downloads the YouGile archive
5. Builds and starts the containers

### Step 4: Access YouGile

```
http://localhost          # with nginx enabled (default)
http://localhost:8001     # with NGINX_ENABLED=false
```

> **Note:** when nginx is enabled, port 8001 is bound to `127.0.0.1` only and is not reachable from the network — all external traffic is routed through nginx on ports 80/443.

### Step 5: Initial configuration

1. Edit `./yougile/conf.json` — set your domain and SMTP settings
2. Restart: `docker compose restart`
3. Set up SSL if needed (see [SSL Certificates](#ssl-certificates))

---

## Re-running the Installer

By default, **all existing files are preserved**. Re-running `./yougile-container install` is safe — it only creates what is missing:

| File                                                | Default behavior     |
| --------------------------------------------------- | -------------------- |
| `conf.json`, `nginx.conf`, `license.key`            | Preserved (`[SKIP]`) |
| `Dockerfile`, `docker-compose.yml`, `.dockerignore` | Preserved (`[SKIP]`) |
| `yougile.tar.gz`                                    | Preserved (`[SKIP]`) |

To force regeneration of all files (for example, after changing `.env`):

```bash
./yougile-container install --regen
```

### Dry run (preview without writing)

The `--dry-run` flag prints the content of all generated files to stdout without writing anything to disk or starting docker-compose:

```bash
./yougile-container install --dry-run
```

Each file is preceded by an `=== filename ===` header, making it easy to filter:

```bash
# View only docker-compose.yml
./yougile-container install --dry-run | grep -A 999 "=== docker-compose.yml ==="

# Compare output across different .env configurations
./yougile-container install --dry-run > before.txt
# ... edit .env ...
./yougile-container install --dry-run > after.txt
diff before.txt after.txt
```

`--dry-run` is useful for code review, IaC integration, and auditing the generated configuration before deployment.

---

## Configuration via .env

`.env.example` is included in the delivery package and documents all supported variables. Copy it to `.env` and set the values you need.

### Installer language

```env
# en (default) or ru
YOUGILE_LANG=en
```

Can also be set via flag: `./yougile-container install --lang=ru`

### Air-gapped / private registry deployment

For deployments without internet access, specify internal addresses:

```env
# YouGile archive — downloaded by the installer before the Docker build
YOUGILE_DOWNLOAD_URL=http://nexus.corp/yougile/latest/yougile.tar.gz

# Docker images — used in Dockerfile and docker-compose.yml
YOUGILE_NODE_IMAGE=registry.corp/node:22.3.0
YOUGILE_NGINX_IMAGE=registry.corp/nginx:alpine
YOUGILE_CERTBOT_IMAGE=registry.corp/certbot:latest
```

> **Important:** `yougile.tar.gz` is downloaded by the installer **before** `docker build`, so the image build itself requires no internet access. Docker base images (`node`, `nginx`) must be pre-loaded into your internal registry separately.

### Nginx

```env
# true (default) — nginx runs as a reverse proxy
# false — nginx is disabled; YouGile is accessible directly on :8001
NGINX_ENABLED=true

# Host ports (change if 80/443 are already in use)
NGINX_HTTP_PORT=80
NGINX_HTTPS_PORT=443
```

### Resource limits

```env
# Leave empty for no limits.
# Memory: 512m, 1g, 2g, etc.  CPU: core count (0.5, 1, 2, etc.)
YOUGILE_MEM_LIMIT=
YOUGILE_CPU_LIMIT=
NGINX_MEM_LIMIT=
NGINX_CPU_LIMIT=
```

When set, a `deploy.resources.limits` section is added to the generated `docker-compose.yml`.

### Archive integrity

```env
# SHA-256 checksum of yougile.tar.gz (hex). Leave empty to skip verification.
# To compute: sha256sum yougile.tar.gz
YOUGILE_CHECKSUM=
```

Verification runs both on download and on subsequent runs when the archive already exists on disk.

### Docker image pinning

For fully reproducible builds, pin images to a specific digest:

```env
YOUGILE_NODE_IMAGE=node:22.3.0@sha256:<digest>
YOUGILE_NGINX_IMAGE=nginx:1.27-alpine@sha256:<digest>
```

To obtain a digest: `docker inspect --format='{{index .RepoDigests 0}}' nginx:alpine`

After any `.env` change, regenerate the configuration:

```bash
./yougile-container install --regen
```

---

## Project Structure

```
./
├── yougile-container          # installer binary
├── .env.example               # configuration template (included in delivery)
├── .env                       # your configuration (created manually from .env.example)
├── yougile.tar.gz             # YouGile archive (downloaded during install)
├── Dockerfile                 # YouGile Docker image definition
├── docker-compose.yml         # container orchestration
├── entrypoint.sh              # Docker entrypoint (privilege drop via gosu)
├── .dockerignore              # Docker build context exclusions
├── yougile/                   # YouGile application data
│   ├── conf.json              # application configuration
│   ├── license.key            # license key
│   ├── logs/                  # application logs
│   ├── user-data/             # user-uploaded files
│   ├── database/              # database files
│   └── extensions/            # extensions
├── nginx/                     # Nginx configuration
│   ├── nginx.conf             # main configuration file
│   ├── conf.d/                # additional configuration snippets
│   └── logs/                  # Nginx logs
└── certbot/                   # SSL certificate management
    ├── www/                   # ACME challenge directory
    ├── conf/                  # certificates and Certbot configuration
    └── README.md              # SSL setup instructions
```

---

## YouGile Configuration

Edit `./yougile/conf.json`:

```json
{
  "port": 8001,
  "mainPageUrl": "https://your-domain.com",
  "emailFrom": "\"YouGile\" <noreply@your-domain.com>",
  "smtp": {
    "host": "smtp.your-provider.com",
    "secure": true,
    "port": 465,
    "auth": {
      "user": "your-email@your-domain.com",
      "pass": "your-password"
    }
  }
}
```

---

## SSL Certificates

Certbot is included as an on-demand tool (`profiles: ["tools"]`) and **does not start automatically**. Nginx starts in HTTP-only mode — the SSL server block in `nginx.conf` is commented out by default.

Any certificate type is supported: corporate CA, Let's Encrypt, self-signed, etc.

### Custom certificate (corporate CA or other)

**Step 1: Place the certificate files**

```bash
mkdir -p ./certbot/conf/live/your-domain.com/

# Full chain (certificate + intermediates)
cat your-cert.crt intermediate.crt > ./certbot/conf/live/your-domain.com/fullchain.pem

# Private key
cp your-private.key ./certbot/conf/live/your-domain.com/privkey.pem

chmod 644 ./certbot/conf/live/your-domain.com/fullchain.pem
chmod 600 ./certbot/conf/live/your-domain.com/privkey.pem
```

**Step 2: (Optional) DH parameters**

```bash
openssl dhparam -out ./certbot/conf/ssl-dhparams.pem 2048
```

**Step 3: Configure Nginx**

Uncomment the HTTPS server block in `./nginx/nginx.conf` and set the certificate paths:

```nginx
ssl_certificate /etc/nginx/ssl/live/your-domain.com/fullchain.pem;
ssl_certificate_key /etc/nginx/ssl/live/your-domain.com/privkey.pem;
```

**Step 4: Validate and restart**

```bash
docker compose exec nginx nginx -t
docker compose restart nginx
```

**Step 5: Update conf.json**

```json
{ "mainPageUrl": "https://your-domain.com" }
```

```bash
docker compose restart yougile
```

### Let's Encrypt (requires public internet access)

```bash
docker compose run --rm certbot certonly --webroot \
  -w /var/www/certbot \
  -d your-domain.com \
  --email your-email@example.com \
  --agree-tos \
  --no-eff-email
```

Then follow steps 3–5 above.

**Auto-renewal:**

```bash
# Add to crontab
0 12 * * * cd /path/to/project && docker compose run --rm certbot renew --quiet && docker compose restart nginx
```

### Verifying SSL

```bash
curl -I https://your-domain.com
openssl s_client -connect your-domain.com:443 -servername your-domain.com
```

---

## Updating YouGile

```bash
./yougile-container update
```

> **Important:** the `yougile` container must be running — the command checks for updates via `docker exec`. If the container is stopped, start it first with `docker compose up -d`.

What the command does:

- Checks for available updates (via the running container)
- Stops the containers
- Downloads the new `yougile.tar.gz`
- Rebuilds and restarts the containers

Configuration and Docker files are **not overwritten**.

To also regenerate Docker files from templates (for example, after changing `.env`):

```bash
./yougile-container update --regen
```

---

## Container Management

```bash
# Stop
docker compose down

# Start
docker compose up -d

# Restart
docker compose restart

# Stream logs
docker compose logs -f
docker compose logs -f yougile
docker compose logs -f nginx

# Container status
docker compose ps

# Resource usage
docker stats
```

### Connecting to containers

```bash
docker compose exec yougile sh
docker compose exec nginx sh
docker compose exec yougile ./server task
```

---

## Data Management

YouGile stores all persistent data in named Docker volumes, which survive `docker compose down` and installer re-runs.

| Volume               | Contents            |
| -------------------- | ------------------- |
| `yougile_database`   | Database files      |
| `yougile_userdata`   | User-uploaded files |
| `yougile_logs`       | System logs         |
| `yougile_extensions` | Extensions          |

### Backup

```bash
docker compose down

BACKUP_DIR="backup-$(date +%Y%m%d)"
mkdir "$BACKUP_DIR"

docker run --rm -v yougile_database:/data   -v "$(pwd)/$BACKUP_DIR":/backup \
  alpine tar czf /backup/database.tar.gz   -C /data .
docker run --rm -v yougile_userdata:/data   -v "$(pwd)/$BACKUP_DIR":/backup \
  alpine tar czf /backup/userdata.tar.gz   -C /data .
docker run --rm -v yougile_logs:/data       -v "$(pwd)/$BACKUP_DIR":/backup \
  alpine tar czf /backup/logs.tar.gz       -C /data .
docker run --rm -v yougile_extensions:/data -v "$(pwd)/$BACKUP_DIR":/backup \
  alpine tar czf /backup/extensions.tar.gz -C /data .

cp -r yougile nginx certbot "$BACKUP_DIR"/

docker compose up -d
```

### Restore from backup

```bash
docker compose down -v  # WARNING: destroys all current volume data

docker volume create yougile_database
docker volume create yougile_userdata
docker volume create yougile_logs
docker volume create yougile_extensions

docker run --rm -v yougile_database:/data   -v "$(pwd)/backup-YYYYMMDD":/backup \
  alpine tar xzf /backup/database.tar.gz   -C /data
docker run --rm -v yougile_userdata:/data   -v "$(pwd)/backup-YYYYMMDD":/backup \
  alpine tar xzf /backup/userdata.tar.gz   -C /data
docker run --rm -v yougile_logs:/data       -v "$(pwd)/backup-YYYYMMDD":/backup \
  alpine tar xzf /backup/logs.tar.gz       -C /data
docker run --rm -v yougile_extensions:/data -v "$(pwd)/backup-YYYYMMDD":/backup \
  alpine tar xzf /backup/extensions.tar.gz -C /data

cp -r backup-YYYYMMDD/yougile backup-YYYYMMDD/nginx backup-YYYYMMDD/certbot .
docker compose up -d
```

### Manual volume export / import

```bash
# List volume contents
docker run --rm -v yougile_database:/data alpine ls -la /data

# Copy FROM volume to host
docker run --rm -v yougile_database:/data -v $(pwd):/backup \
  alpine cp -r /data/. /backup/database/

# Copy TO volume from host (from the out/ directory)
docker compose down
docker run --rm -v yougile_database:/data -v $(pwd)/out:/source \
  alpine sh -c "rm -rf /data/* && cp -r /source/database/. /data/"
docker run --rm -v yougile_userdata:/data -v $(pwd)/out:/source \
  alpine sh -c "rm -rf /data/* && cp -r /source/user-data/. /data/"
docker compose up -d
```

---

## Security

The following hardening measures are applied automatically to the generated `docker-compose.yml` and `Dockerfile`:

| Container | Measures                                                                                                                                                                |
| --------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `yougile` | `security_opt: no-new-privileges:true`, `cap_drop: ALL`, `cap_add: [CHOWN, SETUID, SETGID]`; the entrypoint drops privileges via `gosu node` before starting the server |
| `nginx`   | `security_opt: no-new-privileges:true`, `cap_drop: ALL`, `cap_add: [NET_BIND_SERVICE]`; config files and certificates are mounted read-only (`:ro`)                     |

The `yougile` container starts as root (required to `chown` volume directories and create config symlinks), after which `entrypoint.sh` immediately executes `exec gosu node "$@"` — the server runs as uid 1000 with no privilege escalation path.

**YouGile archive** is downloaded by the installer before the image build — `docker build` requires no internet access. Optional SHA-256 verification is available via `YOUGILE_CHECKSUM`.

**Certbot** (`profiles: ["tools"]`) does not start automatically — it must be invoked explicitly via `docker compose run --rm certbot`.

---

## Troubleshooting

### Port conflicts

If ports 80/443 are already in use, specify alternatives in `.env`:

```env
NGINX_HTTP_PORT=8080
NGINX_HTTPS_PORT=8443
```

Then regenerate the configuration and restart:

```bash
./yougile-container install --regen
```

### File permissions

```bash
sudo chown -R $USER:$USER ./yougile ./nginx ./certbot
chmod -R 755 ./yougile ./nginx ./certbot
```

### YouGile won't start

```bash
docker compose up yougile       # run in foreground to see output directly
docker compose ps               # check health check status
docker compose logs --tail 100 yougile
```

### Nginx diagnostics

```bash
docker compose exec nginx nginx -t          # validate configuration
docker compose exec nginx nginx -s reload   # reload config without restart
docker compose exec nginx curl -v http://yougile:8001  # test backend connectivity
```

### Full rebuild

```bash
docker compose down
docker compose build --no-cache
docker compose up -d
```

### Collecting diagnostics

```bash
echo "=== Docker ===" > debug.txt
docker --version >> debug.txt
docker compose version >> debug.txt
echo "=== Containers ===" >> debug.txt
docker compose ps >> debug.txt
echo "=== Logs ===" >> debug.txt
docker compose logs --tail 200 >> debug.txt

tar -czf support-$(date +%Y%m%d-%H%M%S).tar.gz debug.txt \
  ./yougile/logs ./nginx/logs ./yougile/conf.json ./nginx/nginx.conf docker-compose.yml
```

---

## Additional Resources

- [Docker Compose Reference](https://docs.docker.com/compose/)
- [Nginx Configuration Guide](https://nginx.org/en/docs/)
- [SSL Configuration Generator](https://ssl-config.mozilla.org/)
