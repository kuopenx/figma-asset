package figmaasset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// --- public dispatch ---

func writePNG(platform, outDir, fileName string, exports []PluginExportBytes) ([]string, error) {
	switch platform {
	case PlatformFlutter:
		return writePNGFlutter(outDir, fileName, exports)
	case PlatformAndroid:
		return writePNGAndroid(outDir, fileName, exports)
	case PlatformIOS:
		return writePNGIOS(outDir, fileName, exports)
	case PlatformWeb:
		return writePNGWeb(outDir, fileName, exports)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

func writeSVG(outDir, fileName string, exports []PluginExportBytes) ([]string, error) {
	safeName := sanitizeFileName(fileName)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	files := make([]string, 0, len(exports))
	for _, item := range exports {
		target := filepath.Join(outDir, safeName+".svg")
		if err := bytesToFile(target, item.Bytes); err != nil {
			return nil, err
		}
		files = append(files, target)
	}
	return files, nil
}

// --- Flutter PNG ---

func writePNGFlutter(outDir, fileName string, exports []PluginExportBytes) ([]string, error) {
	safeName := sanitizeFileName(fileName)
	files := make([]string, 0, len(exports))
	for _, item := range exports {
		dir := outDir
		if item.Scale != 1 {
			dir = filepath.Join(outDir, fmt.Sprintf("%.1fx", item.Scale))
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		target := filepath.Join(dir, safeName+".png")
		if err := bytesToFile(target, item.Bytes); err != nil {
			return nil, err
		}
		files = append(files, target)
	}
	return files, nil
}

// --- Android PNG ---

var androidDensityDirs = map[float64]string{
	1:   "drawable-mdpi",
	1.5: "drawable-hdpi",
	2:   "drawable-xhdpi",
	3:   "drawable-xxhdpi",
	4:   "drawable-xxxhdpi",
}

func writePNGAndroid(outDir, fileName string, exports []PluginExportBytes) ([]string, error) {
	safeName := sanitizeFileName(fileName)
	files := make([]string, 0, len(exports))
	for _, item := range exports {
		dirName, ok := androidDensityDirs[item.Scale]
		if !ok {
			return nil, fmt.Errorf("android: no density bucket for scale %v", item.Scale)
		}
		dir := filepath.Join(outDir, dirName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		target := filepath.Join(dir, safeName+".png")
		if err := bytesToFile(target, item.Bytes); err != nil {
			return nil, err
		}
		files = append(files, target)
	}
	return files, nil
}

// --- iOS PNG ---

func writePNGIOS(outDir, fileName string, exports []PluginExportBytes) ([]string, error) {
	safeName := sanitizeFileName(fileName)
	imagesetDir := filepath.Join(outDir, safeName+".imageset")
	if err := os.MkdirAll(imagesetDir, 0o755); err != nil {
		return nil, err
	}

	files := make([]string, 0, len(exports)+1)
	images := make([]iosImage, 0, len(exports))

	for _, item := range exports {
		base := safeName
		if item.Scale != 1 {
			base = safeName + "@" + iosScaleSuffix(item.Scale)
		}
		fname := base + ".png"
		target := filepath.Join(imagesetDir, fname)
		if err := bytesToFile(target, item.Bytes); err != nil {
			return nil, err
		}
		files = append(files, target)
		images = append(images, iosImage{
			Filename: fname,
			Idiom:    "universal",
			Scale:    iosScaleString(item.Scale),
		})
	}

	contents := iosContents{
		Images: images,
		Info:   iosInfo{Author: "figma-asset", Version: 1},
	}
	contentsPath := filepath.Join(imagesetDir, "Contents.json")
	contentsBytes, err := json.MarshalIndent(contents, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(contentsPath, append(contentsBytes, '\n'), 0o644); err != nil {
		return nil, err
	}
	files = append(files, contentsPath)

	return files, nil
}

type iosContents struct {
	Images []iosImage `json:"images"`
	Info   iosInfo    `json:"info"`
}

type iosImage struct {
	Filename string `json:"filename"`
	Idiom    string `json:"idiom"`
	Scale    string `json:"scale"`
}

type iosInfo struct {
	Author  string `json:"author"`
	Version int    `json:"version"`
}

func iosScaleSuffix(scale float64) string {
	if scale == float64(int(scale)) {
		return fmt.Sprintf("%dx", int(scale))
	}
	return fmt.Sprintf("%vx", scale)
}

func iosScaleString(scale float64) string {
	return iosScaleSuffix(scale)
}

// --- Web PNG ---

func writePNGWeb(outDir, fileName string, exports []PluginExportBytes) ([]string, error) {
	safeName := sanitizeFileName(fileName)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	files := make([]string, 0, len(exports))
	for _, item := range exports {
		base := safeName
		if item.Scale != 1 {
			base = safeName + "@" + iosScaleSuffix(item.Scale)
		}
		target := filepath.Join(outDir, base+".png")
		if err := bytesToFile(target, item.Bytes); err != nil {
			return nil, err
		}
		files = append(files, target)
	}
	return files, nil
}

// --- helpers ---

func bytesToFile(target string, raw []int) error {
	data := make([]byte, len(raw))
	for i, v := range raw {
		data[i] = byte(v)
	}
	return os.WriteFile(target, data, 0o644)
}

var invalidFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func sanitizeFileName(input string) string {
	name := strings.TrimSpace(input)
	name = strings.ReplaceAll(name, "-", "_")
	name = invalidFileNameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	name = strings.ToLower(name)
	if name == "" {
		return "figma_asset"
	}
	return name
}
