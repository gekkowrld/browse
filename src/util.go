package src

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Expand user directory (e.g., "~")
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}

// Resolve relative paths to absolute paths
func ResolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return absPath, nil
}

func UniqueSortedEntries(arr []string) []string {
	// Create a map to store unique entries
	uniqueMap := make(map[string]struct{})

	// Add entries to the map
	for _, value := range arr {
		uniqueMap[value] = struct{}{}
	}

	// Extract keys from the map into a slice
	uniqueSlice := make([]string, 0, len(uniqueMap))
	for key := range uniqueMap {
		uniqueSlice = append(uniqueSlice, key)
	}

	// Sort the slice
	sort.Strings(uniqueSlice)

	return uniqueSlice
}

// File extensions and their types (html element?)
// that can be displayed by `most` modern browsers
var viewableFiles = map[string]string{
	".jpg":  "img",
	".jpeg": "img",
	".png":  "img",
	".gif":  "img",
	".bmp":  "img",
	".svg":  "img",
	".webp": "img",
	".pdf":  "pdf",
	".mp3":  "audio",
	".wav":  "audio",
	".mp4":  "video",
	".webm": "video",
	".ogg":  "audio",
	".ico":  "img",
	".tiff": "img",
	".avif": "img",
	".mpeg": "video",
	".mov":  "video",
	".avi":  "video",
	".ts":   "video",
}

func IsViewableInBrowser(filename string) (string, bool) {
	extension := strings.ToLower(filepath.Ext(filename))
	media, ok := viewableFiles[extension]
	return media, ok
}

// isBinary determines if a file at the given path is binary or text.
func IsBinary(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return true
	}
	defer file.Close()

	const maxBufSize = 1024
	buf := make([]byte, maxBufSize)

	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return true
	}

	return !isText(buf[:n])
}

// isText checks if the given byte slice contains only text (not binary).
// From: https://cs.opensource.google/go/x/tools/+/refs/tags/v0.24.0:godoc/util/util.go;l=40
func isText(s []byte) bool {
	const max = 1024 // at least utf8.UTFMax
	if len(s) > max {
		s = s[0:max]
	}
	for i, c := range string(s) {
		if i+utf8.UTFMax > len(s) {
			// last char may be incomplete - ignore
			break
		}
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' && c != '\v' && c != '\r' && c != '\b' {
			// decoding error or control character - not a text file
			return false
		}
	}
	return true
}

func TrimText(text string, at int, end ...bool) string {
	str_len := len(text)
	if str_len < at {
		return text
	}

	var dispEnd bool
	if len(end) >= 1 {
		dispEnd = end[0]
	}

	var ending string = "..."
	if dispEnd {
		// reverse
		text = rev(text)
		cutStr := text[:at]
		return ending + rev(cutStr)
	}
	return text[:at] + ending
}

// From: https://stackoverflow.com/questions/1752414/how-to-reverse-a-string-in-go/1754209#1754209
func rev(input string) string {
	n := 0
	rune := make([]rune, len(input))
	for _, r := range input {
		rune[n] = r
		n++
	}
	rune = rune[0:n]
	// Reverse
	for i := 0; i < n/2; i++ {
		rune[i], rune[n-1-i] = rune[n-1-i], rune[i]
	}
	// Convert back to UTF-8.
	return string(rune)
}
