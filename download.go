package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DownloadYougileArchive downloads yougile.tar.gz to the current directory.
// If force is false and the file already exists, the download is skipped.
// Uses a temp file + rename for an atomic write so a failed download never
// leaves a partial archive behind.
func DownloadYougileArchive(force bool) error {
	const dest = "yougile.tar.gz"
	url := youGileDownloadURL()

	if !force && fileExists(dest) {
		fmt.Println("  [SKIP] " + tr("Архив уже существует, пропускаем загрузку", "Archive already exists, skipping download"))
		return verifyChecksumIfSet(dest)
	}

	fmt.Printf("  "+tr("Загрузка архива из %s ...", "Downloading archive from %s ...")+"\n", url)

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf(tr("ошибка загрузки архива: %v", "failed to download archive: %v"), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(tr("сервер вернул статус %s", "server returned status %s"), resp.Status)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf(tr("не удалось создать временный файл: %v", "failed to create temp file: %v"), err)
	}

	_, copyErr := io.Copy(f, resp.Body)
	f.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return fmt.Errorf(tr("ошибка записи архива: %v", "failed to write archive: %v"), copyErr)
	}

	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return fmt.Errorf(tr("не удалось сохранить архив: %v", "failed to save archive: %v"), err)
	}

	if err := verifyChecksumIfSet(dest); err != nil {
		os.Remove(dest)
		return err
	}

	fmt.Println("  [OK] " + tr("Архив загружен", "Archive downloaded"))
	return nil
}

func verifyChecksumIfSet(path string) error {
	expected := os.Getenv("YOUGILE_CHECKSUM")
	if expected == "" {
		return nil
	}
	fmt.Println("  " + tr("Проверка контрольной суммы...", "Verifying checksum..."))
	if err := verifySHA256(path, expected); err != nil {
		return fmt.Errorf(tr("ошибка проверки контрольной суммы: %v", "checksum verification failed: %v"), err)
	}
	fmt.Println("  [OK] " + tr("Контрольная сумма верна", "Checksum verified"))
	return nil
}

func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf(tr("ожидалось %s, получено %s", "expected %s, got %s"), expected, got)
	}
	return nil
}
