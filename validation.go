package main

import (
	"fmt"
	"os"
)

// RequiredFiles holds the lists of files and directories required for a valid project.
type RequiredFiles struct {
	ConfigFiles []string
	DockerFiles []string
	Directories []string
}

// GetRequiredFiles returns the expected set of files and directories.
func GetRequiredFiles() RequiredFiles {
	dirs := GetProjectDirectories()

	allDirs := make([]string, 0)
	allDirs = append(allDirs, dirs.YouGileDirectories...)
	allDirs = append(allDirs, dirs.NginxDirectories...)
	allDirs = append(allDirs, dirs.CertbotDirectories...)

	return RequiredFiles{
		ConfigFiles: []string{
			"./yougile/license.key",
			"./yougile/conf.json",
			"./nginx/nginx.conf",
			"./certbot/README.md",
		},
		DockerFiles: []string{
			"Dockerfile",
			"entrypoint.sh",
			"docker-compose.yml",
			".dockerignore",
			"yougile.tar.gz",
		},
		Directories: allDirs,
	}
}

// ValidateProjectStructure verifies that all required files and directories exist.
func ValidateProjectStructure() error {
	required := GetRequiredFiles()

	fmt.Println("  " + tr("Проверка конфигурационных файлов...", "Checking configuration files..."))
	if err := validateFiles(required.ConfigFiles); err != nil {
		return err
	}

	fmt.Println("  " + tr("Проверка Docker файлов...", "Checking Docker files..."))
	if err := validateFiles(required.DockerFiles); err != nil {
		return err
	}

	fmt.Println("  " + tr("Проверка структуры каталогов...", "Checking directory structure..."))
	if err := validateDirectories(required.Directories); err != nil {
		return err
	}

	fmt.Println("  [SUCCESS] " + tr("Все файлы и каталоги созданы успешно!", "All files and directories created successfully!"))
	return nil
}

func validateFiles(files []string) error {
	for _, file := range files {
		if err := checkFileExists(file); err != nil {
			return fmt.Errorf(tr("файл не найден: %s (%v)", "file not found: %s (%v)"), file, err)
		}
		fmt.Printf("    [OK] %s\n", file)
	}
	return nil
}

func validateDirectories(directories []string) error {
	for _, dir := range directories {
		if err := checkDirectoryExists(dir); err != nil {
			return fmt.Errorf(tr("каталог не найден: %s (%v)", "directory not found: %s (%v)"), dir, err)
		}
		fmt.Printf("    [OK] %s\n", dir)
	}
	return nil
}

func checkFileExists(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf(tr("файл не существует", "file does not exist"))
	}
	if err != nil {
		return fmt.Errorf(tr("ошибка доступа к файлу: %v", "error accessing file: %v"), err)
	}
	if info.IsDir() {
		return fmt.Errorf(tr("ожидался файл, но найден каталог", "expected a file but found a directory"))
	}
	return nil
}

func checkDirectoryExists(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf(tr("каталог не существует", "directory does not exist"))
	}
	if err != nil {
		return fmt.Errorf(tr("ошибка доступа к каталогу: %v", "error accessing directory: %v"), err)
	}
	if !info.IsDir() {
		return fmt.Errorf(tr("ожидался каталог, но найден файл", "expected a directory but found a file"))
	}
	return nil
}
