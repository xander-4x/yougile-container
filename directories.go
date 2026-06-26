package main

import (
	"fmt"
	"os"
)

// ProjectDirectories holds the directory lists for each subsystem.
type ProjectDirectories struct {
	YouGileDirectories []string
	NginxDirectories   []string
	CertbotDirectories []string
}

// GetProjectDirectories returns the full set of directories to create.
func GetProjectDirectories() ProjectDirectories {
	return ProjectDirectories{
		YouGileDirectories: []string{
			"./yougile",
		},
		NginxDirectories: []string{
			"./nginx/conf.d",
			"./nginx/logs",
		},
		CertbotDirectories: []string{
			"./certbot/www",
			"./certbot/conf",
			"./certbot/conf/live",
			"./certbot/conf/archive",
			"./certbot/conf/renewal",
		},
	}
}

// CreateDirectoryStructure creates the full directory tree for the project.
func CreateDirectoryStructure() error {
	dirs := GetProjectDirectories()

	allDirs := make([]string, 0)
	allDirs = append(allDirs, dirs.YouGileDirectories...)
	allDirs = append(allDirs, dirs.NginxDirectories...)
	allDirs = append(allDirs, dirs.CertbotDirectories...)

	for _, dir := range allDirs {
		if err := createDirectory(dir); err != nil {
			return fmt.Errorf(tr("не удалось создать каталог %s: %v", "failed to create directory %s: %v"), dir, err)
		}
		fmt.Printf("  %s\n", dir)
	}

	if err := createCertbotReadme(); err != nil {
		return fmt.Errorf(tr("не удалось создать README для Certbot: %v", "failed to create Certbot README: %v"), err)
	}

	return nil
}

func createDirectory(path string) error {
	if dryRun {
		return nil
	}
	return os.MkdirAll(path, 0775)
}

func createCertbotReadme() error {
	ruContent := `# SSL сертификаты

Эта папка содержит SSL сертификаты для HTTPS доступа к YouGile.

## Структура каталогов:
- **www/**: веб-корень для ACME challenge (для Let's Encrypt)
- **conf/**: SSL сертификаты и конфигурация
- **conf/live/your-domain.com/**: актуальные сертификаты для вашего домена
- **conf/archive/**: архив сертификатов (для Let's Encrypt)
- **conf/renewal/**: конфигурация автообновления (для Let's Encrypt)

## Вариант 1: Использование собственных SSL сертификатов (рекомендуется)

Если у вас есть собственные SSL сертификаты:

### Шаг 1: Разместите ваши сертификаты

` + "```bash" + `
# Создайте директорию для вашего домена
mkdir -p ./conf/live/your-domain.com/

# Скопируйте сертификат (с цепочкой)
cat your-certificate.crt intermediate.crt > ./conf/live/your-domain.com/fullchain.pem

# Или если у вас уже есть fullchain:
cp your-fullchain.pem ./conf/live/your-domain.com/fullchain.pem

# Скопируйте приватный ключ
cp your-private.key ./conf/live/your-domain.com/privkey.pem

# Установите правильные права
chmod 644 ./conf/live/your-domain.com/fullchain.pem
chmod 600 ./conf/live/your-domain.com/privkey.pem
` + "```" + `

### Шаг 2: Создайте DH параметры

` + "```bash" + `
openssl dhparam -out ./conf/ssl-dhparams.pem 2048
` + "```" + `

### Шаг 3: Настройте Nginx

Отредактируйте ` + "`../nginx/nginx.conf`" + ` и укажите пути к сертификатам:
` + "```nginx" + `
ssl_certificate /etc/nginx/ssl/live/your-domain.com/fullchain.pem;
ssl_certificate_key /etc/nginx/ssl/live/your-domain.com/privkey.pem;
ssl_dhparam /etc/nginx/ssl/ssl-dhparams.pem;
` + "```" + `

### Шаг 4: Перезапустите Nginx

` + "```bash" + `
docker compose restart nginx
` + "```" + `

## Вариант 2: Автоматическое получение через Let's Encrypt

Если у вас нет собственных сертификатов:

### Получение нового сертификата:
` + "```bash" + `
docker compose run --rm certbot certonly --webroot \
  -w /var/www/certbot \
  -d your-domain.com \
  --email your-email@example.com \
  --agree-tos \
  --no-eff-email
` + "```" + `

### Проверка статуса сертификатов:
` + "```bash" + `
docker compose run --rm certbot certificates
` + "```" + `

### Тестовое обновление:
` + "```bash" + `
docker compose run --rm certbot renew --dry-run
` + "```" + `

### Ручное обновление:
` + "```bash" + `
docker compose run --rm certbot renew
docker compose restart nginx
` + "```" + `

### Автоматическое обновление (cron):
` + "```bash" + `
0 12 * * * cd /path/to/your/project && docker compose run --rm certbot renew --quiet && docker compose restart nginx
` + "```" + `

## Проверка сертификатов

` + "```bash" + `
# Проверка срока действия
openssl x509 -in ./conf/live/your-domain.com/fullchain.pem -noout -dates

# Проверка содержимого
openssl x509 -in ./conf/live/your-domain.com/fullchain.pem -noout -text

# Проверка приватного ключа
openssl rsa -in ./conf/live/your-domain.com/privkey.pem -check
` + "```" + `

## Важные замечания

- Сертификаты монтируются в Nginx контейнер по пути ` + "`/etc/nginx/ssl/`" + `
- При замене сертификатов всегда перезапускайте Nginx
- Храните приватные ключи в безопасности (права 600)
- Fullchain должен содержать как ваш сертификат, так и промежуточные сертификаты
`

	enContent := `# SSL Certificates

This folder contains SSL certificates for HTTPS access to YouGile.

## Directory structure:
- **www/**: Web root for ACME challenge (for Let's Encrypt)
- **conf/**: SSL certificates and configuration
- **conf/live/your-domain.com/**: Active certificates for your domain
- **conf/archive/**: Certificate archive (for Let's Encrypt)
- **conf/renewal/**: Auto-renewal configuration (for Let's Encrypt)

## Option 1: Using your own SSL certificates (recommended)

If you have your own SSL certificates:

### Step 1: Place your certificates

` + "```bash" + `
# Create the directory for your domain
mkdir -p ./conf/live/your-domain.com/

# Copy the certificate (with chain)
cat your-certificate.crt intermediate.crt > ./conf/live/your-domain.com/fullchain.pem

# Or if you already have a fullchain file:
cp your-fullchain.pem ./conf/live/your-domain.com/fullchain.pem

# Copy the private key
cp your-private.key ./conf/live/your-domain.com/privkey.pem

# Set correct permissions
chmod 644 ./conf/live/your-domain.com/fullchain.pem
chmod 600 ./conf/live/your-domain.com/privkey.pem
` + "```" + `

### Step 2: Generate DH parameters

` + "```bash" + `
openssl dhparam -out ./conf/ssl-dhparams.pem 2048
` + "```" + `

### Step 3: Configure Nginx

Edit ` + "`../nginx/nginx.conf`" + ` and set the certificate paths:
` + "```nginx" + `
ssl_certificate /etc/nginx/ssl/live/your-domain.com/fullchain.pem;
ssl_certificate_key /etc/nginx/ssl/live/your-domain.com/privkey.pem;
ssl_dhparam /etc/nginx/ssl/ssl-dhparams.pem;
` + "```" + `

### Step 4: Restart Nginx

` + "```bash" + `
docker compose restart nginx
` + "```" + `

## Option 2: Automatic certificate via Let's Encrypt

If you don't have your own certificates:

### Obtain a new certificate:
` + "```bash" + `
docker compose run --rm certbot certonly --webroot \
  -w /var/www/certbot \
  -d your-domain.com \
  --email your-email@example.com \
  --agree-tos \
  --no-eff-email
` + "```" + `

### Check certificate status:
` + "```bash" + `
docker compose run --rm certbot certificates
` + "```" + `

### Test renewal:
` + "```bash" + `
docker compose run --rm certbot renew --dry-run
` + "```" + `

### Manual renewal:
` + "```bash" + `
docker compose run --rm certbot renew
docker compose restart nginx
` + "```" + `

### Automatic renewal (cron):
` + "```bash" + `
0 12 * * * cd /path/to/your/project && docker compose run --rm certbot renew --quiet && docker compose restart nginx
` + "```" + `

## Certificate verification

` + "```bash" + `
# Check expiry date
openssl x509 -in ./conf/live/your-domain.com/fullchain.pem -noout -dates

# Check contents
openssl x509 -in ./conf/live/your-domain.com/fullchain.pem -noout -text

# Check private key
openssl rsa -in ./conf/live/your-domain.com/privkey.pem -check
` + "```" + `

## Important notes

- Certificates are mounted in the Nginx container at ` + "`/etc/nginx/ssl/`" + `
- Always restart Nginx after replacing certificates
- Keep private keys secure (permissions 600)
- Fullchain must contain both your certificate and any intermediate certificates
`

	readmeContent := tr(ruContent, enContent)
	fmt.Printf("  ./certbot/README.md\n")
	return writeFile("./certbot/README.md", readmeContent)
}
