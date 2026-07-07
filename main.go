package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	initFlags() // parse --dry-run / --regen before anything else
	writeEnvExample()
	loadDotEnv()
	initLang()

	args := filterArgs(os.Args[1:])

	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "install":
		handleInstall()
	case "update":
		if dryRun {
			fmt.Fprintln(os.Stderr, tr("Флаг --dry-run поддерживается только командой install",
				"--dry-run is only supported with the install command"))
			os.Exit(1)
		}
		if err := handleUpdate(); err != nil {
			fmt.Printf("[ERROR] "+tr("Ошибка обновления: %v\n", "Update failed: %v\n"), err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, tr("Неизвестная команда: %s\n\n", "Unknown command: %s\n\n"), args[0])
		printUsage()
		os.Exit(1)
	}
}

func handleInstall() {
	if dryRun {
		fmt.Println(tr("[DRY-RUN] Предпросмотр генерируемых файлов (ничего не записывается на диск):",
			"[DRY-RUN] Preview of generated files (nothing is written to disk):"))
		fmt.Println()
	}

	fmt.Println(tr("Инициализация проекта YouGile...", "Initializing YouGile project..."))

	fmt.Println(tr("Создание структуры каталогов...", "Creating directory structure..."))
	if err := CreateDirectoryStructure(); err != nil {
		fmt.Printf("[ERROR] "+tr("Ошибка создания каталогов: %v\n", "Failed to create directories: %v\n"), err)
		os.Exit(1)
	}

	fmt.Println(tr("Создание конфигурационных файлов...", "Creating configuration files..."))
	if err := CreateConfigurationFiles(); err != nil {
		fmt.Printf("[ERROR] "+tr("Ошибка создания конфигурационных файлов: %v\n", "Failed to create configuration files: %v\n"), err)
		os.Exit(1)
	}

	fmt.Println(tr("Создание Docker конфигурации...", "Creating Docker configuration..."))
	if err := CreateDockerFiles(); err != nil {
		fmt.Printf("[ERROR] "+tr("Ошибка создания Docker файлов: %v\n", "Failed to create Docker files: %v\n"), err)
		os.Exit(1)
	}

	if dryRun {
		fmt.Println(tr("\n[DRY-RUN] Загрузка архива пропущена.", "\n[DRY-RUN] Archive download skipped."))
		fmt.Println(tr("[DRY-RUN] Валидация и запуск docker-compose пропущены.", "[DRY-RUN] Validation and docker-compose start skipped."))
		return
	}

	fmt.Println(tr("Загрузка архива YouGile...", "Downloading YouGile archive..."))
	if err := DownloadYougileArchive(false); err != nil {
		fmt.Printf("[ERROR] "+tr("Ошибка загрузки архива: %v\n", "Failed to download archive: %v\n"), err)
		os.Exit(1)
	}

	fmt.Println(tr("Проверка созданных файлов и каталогов...", "Validating created files and directories..."))
	if err := ValidateProjectStructure(); err != nil {
		fmt.Printf("[ERROR] "+tr("Ошибка валидации структуры проекта: %v\n", "Project structure validation failed: %v\n"), err)
		os.Exit(1)
	}

	fmt.Println(tr("Запуск docker-compose...", "Starting docker-compose..."))
	if err := runDockerCompose(); err != nil {
		fmt.Printf("[ERROR] "+tr("Ошибка запуска docker-compose: %v\n", "Failed to start docker-compose: %v\n"), err)
		os.Exit(1)
	}

	printSuccessMessage()
}

func printUsage() {
	fmt.Println(tr("YouGile Container — инструмент развёртывания YouGile в Docker",
		"YouGile Container — YouGile Docker deployment tool"))
	fmt.Println()
	fmt.Println(tr("Использование:", "Usage:"))
	fmt.Println("  yougile-container <command> [flags]")
	fmt.Println()
	fmt.Println(tr("Команды:", "Commands:"))
	fmt.Println("  install    " + tr("Развернуть YouGile (первичная установка)", "Deploy YouGile (initial setup)"))
	fmt.Println("  update     " + tr("Обновить YouGile до последней версии", "Update YouGile to the latest version"))
	fmt.Println()
	fmt.Println(tr("Флаги:", "Flags:"))
	fmt.Println("  --regen      " + tr("Пересоздать конфигурационные файлы даже если они уже существуют", "Regenerate config files even if they already exist"))
	fmt.Println("  --dry-run    " + tr("Вывести генерируемые файлы в stdout без записи на диск (только install)", "Print generated files to stdout without writing to disk (install only)"))
	fmt.Println("  --lang=en    " + tr("Язык вывода установщика: en (по умолчанию) или ru", "Installer output language: en (default) or ru"))
	fmt.Println("  --lang=ru")
	fmt.Println()
	fmt.Println(tr("Примеры:", "Examples:"))
	fmt.Println("  yougile-container install")
	fmt.Println("  yougile-container install --regen")
	fmt.Println("  yougile-container install --dry-run")
	fmt.Println("  yougile-container update")
	fmt.Println("  yougile-container update --regen")
}

// regenConfigs forces Docker/config files to be regenerated even if they already exist.
var regenConfigs bool

// dryRun prints generated file content to stdout without writing anything to disk.
var dryRun bool

func initFlags() {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--regen":
			regenConfigs = true
		case "--dry-run":
			dryRun = true
			regenConfigs = true // show all content, never [SKIP]
		}
	}
}

// filterArgs strips known flags so subcommand detection still works
// regardless of flag order (e.g. "--lang=en update" or "update --regen").
func filterArgs(args []string) []string {
	skip := map[string]bool{
		"--lang=en": true, "--en": true,
		"--lang=ru": true, "--ru": true,
		"--regen":   true,
		"--dry-run": true,
	}
	result := make([]string, 0, len(args))
	for _, arg := range args {
		if !skip[arg] {
			result = append(result, arg)
		}
	}
	return result
}

func runDockerCompose() error {
	// --remove-orphans cleans up containers of services removed from the
	// compose file (e.g. nginx/certbot after switching NGINX_ENABLED=false).
	cmd := exec.Command(containerRuntime(), "compose", "up", "-d", "--build", "--remove-orphans")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func printSuccessMessage() {
	url := "http://localhost"
	if !nginxEnabled() {
		url = "http://localhost:8001"
	} else if p := nginxHTTPPort(); p != "80" {
		url = "http://localhost:" + p
	}

	fmt.Println("\n[SUCCESS] " + tr("Проект YouGile успешно инициализирован!", "YouGile project successfully initialized!"))
	fmt.Println("\n" + tr("Полезная информация:", "Useful information:"))
	fmt.Println("   " + tr("YouGile доступен по адресу:", "YouGile is available at:") + " " + url)
	fmt.Println("   " + tr("Настройте SMTP в файле", "Configure SMTP in") + " ./yougile/conf.json")
	fmt.Println("   " + tr("Для получения SSL сертификата выполните:", "To obtain an SSL certificate run:"))
	fmt.Println("      docker compose run --rm certbot certonly --webroot -w /var/www/certbot \\")
	fmt.Println("        -d your-domain.com --email your-email@example.com --agree-tos --no-eff-email")
	fmt.Println("   " + tr("После получения SSL обновите nginx.conf для HTTPS", "After obtaining SSL update nginx.conf for HTTPS"))
	fmt.Println("\n" + tr("Управление проектом:", "Project management:"))
	fmt.Println("   - " + tr("Остановить:", "Stop:") + "     docker compose down")
	fmt.Println("   - " + tr("Просмотр логов:", "View logs:") + " docker compose logs -f")
	fmt.Println("   - " + tr("Перезапустить:", "Restart:") + "  docker compose restart")
	fmt.Println("   - " + tr("Переустановить (сохранить конфиги):", "Re-install (keep configs):") + "    ./yougile-container install")
	fmt.Println("   - " + tr("Переустановить + пересоздать конфиги:", "Re-install + regen configs:") + " ./yougile-container install --regen")
	fmt.Println("   - " + tr("Предпросмотр конфигов (без записи):", "Preview generated files (dry run):") + " ./yougile-container install --dry-run")
	fmt.Println("   - " + tr("Обновить:", "Update:") + "                                    ./yougile-container update")
	fmt.Println("   - " + tr("Обновить + пересоздать конфиги:", "Update + regen configs:") + "           ./yougile-container update --regen")
}

func handleUpdate() error {
	fmt.Println(tr("Проверка обновлений YouGile...", "Checking for YouGile updates..."))

	updateAvailable, err := checkForUpdates()
	if err != nil {
		return fmt.Errorf(tr("ошибка проверки обновлений: %v", "update check failed: %v"), err)
	}

	if !updateAvailable {
		fmt.Println("[SUCCESS] " + tr("Обновления не требуются. Текущая версия актуальна.", "No updates required. Current version is up to date."))
		return nil
	}

	fmt.Println(tr("Обнаружено обновление! Начинаю процесс обновления...", "Update detected! Starting update process..."))

	fmt.Println(tr("Остановка контейнеров...", "Stopping containers..."))
	if err := stopContainers(); err != nil {
		return fmt.Errorf(tr("ошибка остановки контейнеров: %v", "failed to stop containers: %v"), err)
	}

	if regenConfigs {
		fmt.Println(tr("Пересоздание Docker конфигурации...", "Regenerating Docker configuration..."))
		if err := CreateDockerFiles(); err != nil {
			return fmt.Errorf(tr("ошибка создания Docker файлов: %v", "failed to create Docker files: %v"), err)
		}
	}

	fmt.Println(tr("Загрузка новой версии YouGile...", "Downloading new YouGile version..."))
	if err := DownloadYougileArchive(true); err != nil {
		return fmt.Errorf(tr("ошибка загрузки архива: %v", "failed to download archive: %v"), err)
	}

	fmt.Println(tr("Запуск обновленных контейнеров...", "Starting updated containers..."))
	if err := runDockerCompose(); err != nil {
		return fmt.Errorf(tr("ошибка запуска docker-compose: %v", "failed to start docker-compose: %v"), err)
	}

	fmt.Println("[SUCCESS] " + tr("Обновление завершено успешно!", "Update completed successfully!"))
	return nil
}

func checkForUpdates() (bool, error) {
	// compose exec resolves the service by name, so this keeps working
	// even if container_name is changed or removed in the compose file.
	cmd := exec.Command(containerRuntime(), "compose", "exec", "-T", "yougile", "./server", "task", "show-updates")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf(tr("ошибка выполнения команды проверки обновлений: %v", "failed to run update check command: %v"), err)
	}

	outputStr := string(output)
	fmt.Println(tr("Результат проверки обновлений:", "Update check result:"))
	fmt.Println(outputStr)

	if strings.Contains(outputStr, "Available updates:") {
		return true, nil
	}

	if strings.Contains(outputStr, "Already up to date") {
		return false, nil
	}

	return false, nil
}

func stopContainers() error {
	cmd := exec.Command(containerRuntime(), "compose", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
