package main

import (
	"fmt"
	"os"
)

// CreateConfigurationFiles creates all required configuration files.
func CreateConfigurationFiles() error {
	configFiles := []struct {
		name     string
		path     string
		function func() error
	}{
		{tr("YouGile конфигурация", "YouGile configuration"), "./yougile/conf.json", createYougileConfig},
		{tr("Nginx конфигурация", "Nginx configuration"), "./nginx/nginx.conf", createNginxConfig},
		{tr("Демо лицензия", "Demo license"), "./yougile/license.key", createDemoLicense},
	}

	for _, file := range configFiles {
		fmt.Printf("  %s...\n", file.name)
		if err := file.function(); err != nil {
			return fmt.Errorf(tr("ошибка создания %s: %v", "error creating %s: %v"), file.name, err)
		}
	}

	return nil
}

// createYougileConfig writes the YouGile application config to ./yougile/conf.json.
// Never overwrites an existing file (it holds user edits such as SMTP
// credentials), even with --regen. Dry-run always previews the template.
func createYougileConfig() error {
	if !dryRun && fileExists("./yougile/conf.json") {
		fmt.Println("    [SKIP] " + tr("Файл уже существует, пропускаем", "File already exists, skipping"))
		return nil
	}

	configContent := `{
  /**
   * Port that is used by YouGile backend.
   * It is recommended to use nginx
   * in front of YouGile with https turned on.
   * If YouGile is the only app served on the server,
   * you can run "yougile setup:nginx" to install default nginx
   * configurations for YouGile
   * (this command rewrites existent nginx configurations).
   */
  "port": 8001,

  /**
   * URL that is used to access your YouGile server.
   * Update this to your actual domain when you get SSL certificate.
   */
  "mainPageUrl": "https://yougile.example.com",

  /**
   * Email address that will be used as sender for all outgoing emails.
   */
  "emailFrom": "\"YouGile\" <noreply@example.com>",

  /**
   * Logging configuration.
   * Logs will be saved to the mounted volume ./yougile/logs/
   */
  "logStreams": [
    {"level": "error", "path": "/opt/yougile/logs/error.log"},
    {"level": "info", "path": "/opt/yougile/logs/info.log"}
  ],

  /**
   * SMTP settings for sending emails.
   * Update these settings with your SMTP provider details.
   */
  "smtp": {
    "host": "smtp.example.com",
    "secure": true,
    "port": 465,
    "pool": true,
    "rateLimit": 5,
    "maxConnections": 20,
    "auth": {
      "user": "your-email@example.com",
      "pass": "YOUR_SMTP_PASSWORD"
    }
  }
}`

	return writeFile("./yougile/conf.json", configContent)
}

// createNginxConfig writes the Nginx reverse proxy config to ./nginx/nginx.conf.
// Never overwrites an existing file (it holds user edits such as SSL paths),
// even with --regen. Dry-run always previews the template.
func createNginxConfig() error {
	if !dryRun && fileExists("./nginx/nginx.conf") {
		fmt.Println("    [SKIP] " + tr("Файл уже существует, пропускаем", "File already exists, skipping"))
		return nil
	}

	nginxConfig := `events {
    worker_connections 1024;
    multi_accept on;
    use epoll;
}

http {
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 120;
    types_hash_max_size 2048;

    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    gzip on;

    server {
        listen 80;
        # server_name <YOUR_SERVER_NAME>;

        client_max_body_size 50M;
        client_body_buffer_size 50M;

        # ACME challenge for Let's Encrypt (certbot webroot)
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }

        # Plain HTTP proxy to YouGile. After enabling the HTTPS server
        # block below, replace this location with a redirect:
        #     location / { return 301 https://$host$request_uri; }
        location / {
            proxy_pass http://yougile:8001;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $remote_addr;
            proxy_set_header Host $http_host;
            proxy_read_timeout 1h;
            proxy_connect_timeout 1h;
            proxy_send_timeout 1h;
            proxy_pass_header Server;
            proxy_max_temp_file_size 0;
        }
    }

#     server {
#         listen 443 ssl;
#         server_name <YOUR_SERVER_NAME>;

#         ssl_protocols TLSv1.2;
#         ssl_certificate <PATH_TO_CERT>;
#         ssl_certificate_key <PATH_TO_KEY>;
#         ssl_dhparam <PATH_TO_DHPARAM>;

#         ssl_ciphers ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-SHA384:ECDHE-RSA-AES256-SHA:DHE-RSA-AES256-SHA;
#         ssl_prefer_server_ciphers on;

#         ssl_session_timeout 10m;
#         ssl_session_cache shared:SSL:10m;

#         client_max_body_size 50M;
#         client_body_buffer_size 50M;

#         gzip on;
#         gzip_http_version 1.1;
#         gzip_comp_level 5;
#         gzip_min_length 4096;
#         gzip_proxied any;
#         gzip_types text/plain text/xml text/css application/x-javascript application/javascript application/json application/x-font-ttf;
#         gzip_vary on;

#         add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
#         add_header X-Frame-Options SAMEORIGIN;
#         add_header X-XSS-Protection "1; mode=block";
#         add_header X-Content-Type-Options nosniff;

# # Remove this block if you use the restrictUserDataAccess parameter
# # (user-data files are served directly by nginx from the read-only
# # yougile_userdata volume mount; @local-data proxies cache misses to YouGile)
#         location /user-data/ {
#             add_header X-YouGile-Served data;
#             add_header 'Content-Disposition' 'attachment';

#             location ~* \.(jpe?g|png|pdf|gif|mp4|m4p|mp3|avi|wmv)$ {
#                 add_header 'Content-Disposition' '';
#                 root /opt/yougile;
#                 try_files $uri @local-data;
#             }

#             root /opt/yougile;
#             try_files $uri @local-data;
#         }

#         location @local-data {
#             proxy_pass http://yougile:8001;
#             proxy_http_version 1.1;
#             proxy_set_header X-Real-IP $remote_addr;
#             proxy_set_header X-Forwarded-For $remote_addr;
#             proxy_set_header Host $http_host;
#             proxy_pass_header Server;
#         }
# # End of block to remove

#         location ~ /\. {
#             deny all;
#         }

#         location / {
#             proxy_pass http://yougile:8001;
#             proxy_http_version 1.1;
#             proxy_set_header Upgrade $http_upgrade;
#             proxy_set_header Connection "upgrade";
#             proxy_set_header X-Real-IP $remote_addr;
#             proxy_set_header X-Forwarded-For $remote_addr;
#             proxy_set_header Host $http_host;
#             proxy_read_timeout 1h;
#             proxy_connect_timeout 1h;
#             proxy_send_timeout 1h;
#             proxy_pass_header Server;
#             proxy_max_temp_file_size 0;
#         }
#     }
}`

	return writeFile("./nginx/nginx.conf", nginxConfig)
}

// createDemoLicense writes the demo license key to ./yougile/license.key.
func createDemoLicense() error {
	if !dryRun && fileExists("./yougile/license.key") {
		fmt.Println("    [SKIP] " + tr("Файл уже существует, пропускаем", "File already exists, skipping"))
		return nil
	}

	return writeFile("./yougile/license.key", "demo-platform-license")
}

// fileExists reports whether path exists (file or directory).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// writeFile writes content to path, or prints it to stdout when --dry-run is active.
func writeFile(path, content string) error {
	if dryRun {
		fmt.Printf("\n=== %s ===\n%s\n", path, content)
		return nil
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(tr("не удалось создать файл %s: %v", "failed to create file %s: %v"), path, err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf(tr("не удалось записать в файл %s: %v", "failed to write to file %s: %v"), path, err)
	}

	return nil
}
