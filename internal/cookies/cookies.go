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

	// Cookie files are optional.
	// They can be downloaded through CookiesLink if configured.
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

func downloadCookieFile(url string) error {
	id := filepath.Base(url)

	if id == "" || id == "." || id == "/" {
		return fmt.Errorf("invalid cookie URL: %s", url)
	}

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

	for _, file := range files {
		if filepath.Base(file) == "example.txt" {
			continue
		}

		filtered = append(filtered, file)
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
		gologging.WarnF(
			"Failed to load cookie cache: %v",
			err,
		)
		return "", err
	}

	if len(cachedFiles) == 0 {
		gologging.Warn("No cookie files available")
		return "", nil
	}

	return cachedFiles[rand.Intn(len(cachedFiles))], nil
}
