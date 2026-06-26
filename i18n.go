package main

import "os"

// langEn true → English output; false → Russian output. Default: English.
var langEn = true

func initLang() {
	// CLI flags take highest priority.
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--lang=en", "--en":
			langEn = true
			return
		case "--lang=ru", "--ru":
			langEn = false
			return
		}
	}

	// Fall back to YOUGILE_LANG env var (set from .env or shell).
	switch os.Getenv("YOUGILE_LANG") {
	case "ru":
		langEn = false
	case "en":
		langEn = true
	// default: leave langEn = true (English)
	}
}

// tr returns en when English mode is active, otherwise ru.
func tr(ru, en string) string {
	if langEn {
		return en
	}
	return ru
}
