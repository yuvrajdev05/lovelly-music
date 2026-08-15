/*
 * ● ArcMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Arc
 */

package cookies

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Laky-64/gologging"
	"resty.dev/v3"

	"main/internal/config"
)

const cookieDir = "internal/cookies"

var (
	cachedFiles []string
	cacheOnce   sync.Once
	client      = resty.New()
)

func init() {
	gologging.Debug("🔹 Initializing cookies...")

	if err := copyEmbeddedCookies(); err != nil {
		gologging.Fatal("Failed to copy embedded cookies:", err)
	}

	urls := strings.Fields(config.CookiesLink)
	for _, url := range urls {
		if err := downloadCookieFile(url); err != nil {
			gologging.WarnF(
				"Failed to download cookie file from %s: %v",
				url,
				err,
			)
		}
	}
}

// copyEmbeddedCookies is intentionally a no-op.
// Cookie files are optional and can be supplied through CookiesLink.
// Keeping cookie data out of the Go embed pattern also makes Docker builds
// independent of whether optional .txt cookie files are present in the build context.
func copyEmbeddedCookies() error {
	return nil
}

func downloadCookieFile(url string) error {
	id := filepath.Base(url)
	rawURL := "https://batbin.me/raw/" + id
	filePath := filepath.Join(cookieDir, id+".txt")

	resp, err := client.R().
		SetOutputFileName(filePath).
		Get(rawURL)
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf(
			"unexpected status %d from %s",
			resp.StatusCode(),
			rawURL,
		)
	}

	return nil
}

func loadCookieCache() error {
	files, err := filepath.Glob(filepath.Join(cookieDir, "*.txt"))
	if err != nil {
		return err
	}

	var filtered []string

	for _, f := range files {
		if filepath.Base(f) == "example.txt" {
			continue
		}
		filtered = append(filtered, f)
	}

	cachedFiles = filtered
	return nil
}

func GetRandomCookieFile() (string, error) {
	var err error

	cacheOnce.Do(func() {
		err = loadCookieCache()
	})

	if err != nil {
		gologging.WarnF("Failed to load cookie cache: %v", err)
		return "", err
	}

	if len(cachedFiles) == 0 {
		gologging.Warn("No cookie files available")
		return "", nil
	}

	return cachedFiles[rand.Intn(len(cachedFiles))], nil
}
