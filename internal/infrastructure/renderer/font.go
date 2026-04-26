package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var (
	fontCache   = make(map[string]font.Face)
	fontCacheMu sync.RWMutex
)

// fontDirs lists common system font directories.
var fontDirs = []string{
	"/usr/share/fonts/TTF",
	"/usr/share/fonts/truetype",
	"/usr/share/fonts",
	"/usr/local/share/fonts",
}

// parseFontString parses a font string like "DejaVu Sans 16" or "Monospace 12"
// into a font name and size.
func parseFontString(s string) (name string, size int) {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return "", 0
	}

	// Last part should be the size
	sizeStr := parts[len(parts)-1]
	if sz, err := strconv.Atoi(sizeStr); err == nil {
		name = strings.Join(parts[:len(parts)-1], " ")
		return name, sz
	}

	// No size found, return default
	return s, 0
}

// findFontFile searches for a TTF file matching the given name.
func findFontFile(name string) (string, error) {
	searchName := strings.ToLower(strings.ReplaceAll(name, " ", ""))

	for _, dir := range fontDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !strings.HasSuffix(strings.ToLower(entry.Name()), ".ttf") {
				continue
			}
			fileName := strings.ToLower(strings.TrimSuffix(entry.Name(), ".ttf"))
			fileName = strings.ReplaceAll(fileName, "-", "")
			fileName = strings.ReplaceAll(fileName, " ", "")

			// Match if search name is contained in file name
			if strings.Contains(fileName, searchName) || strings.Contains(searchName, fileName) {
				return filepath.Join(dir, entry.Name()), nil
			}
		}
	}

	return "", fmt.Errorf("font %q not found", name)
}

// loadFont loads a TrueType font face at the given size.
func loadFont(name string, size int) (font.Face, error) {
	if size <= 0 {
		size = 12
	}

	cacheKey := fmt.Sprintf("%s:%d", name, size)

	fontCacheMu.RLock()
	if face, ok := fontCache[cacheKey]; ok {
		fontCacheMu.RUnlock()
		return face, nil
	}
	fontCacheMu.RUnlock()

	fontPath, err := findFontFile(name)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("reading font file: %w", err)
	}

	ft, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing font: %w", err)
	}

	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     96,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("creating font face: %w", err)
	}

	fontCacheMu.Lock()
	fontCache[cacheKey] = face
	fontCacheMu.Unlock()

	return face, nil
}
