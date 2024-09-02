package src

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Expand user directory (e.g., "~")
func expandPath(path string) (string, error) {
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
func resolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return absPath, nil
}

func uniqueSortedEntries(arr []string) []string {
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

var viewableFiles = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".svg":  true,
	".webp": true,
	".pdf":  true,
	".mp3":  true,
	".wav":  true,
	".mp4":  true,
	".webm": true,
	".ogg":  true,
	".ico":  true,
}

func isViewableInBrowser(filename string) bool {
	extension := strings.ToLower(filepath.Ext(filename))
	return viewableFiles[extension]
}

func cFilesWithDetails(dir string) []FileDetail {
	var details []FileDetail
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Println(err)
		return details
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			log.Println(err)
			continue
		}
		if file.IsDir() {
			details = append(details, FileDetail{
				Name:    file.Name(),
				IsDir:   file.IsDir(),
				ModTime: info.ModTime(),
			})
		} else {
			details = append(details, FileDetail{
				Name:    file.Name(),
				IsDir:   file.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}
	}

	return details
}

// List files in the given directory
func cFiles(dir string) map[string]bool {
	lis := make(map[string]bool)
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Println(err)
		return lis
	}

	for _, file := range files {
		fullPath := filepath.Join(dir, file.Name())
		/*if fullPath == dir {
			continue
		}*/
		lis[fullPath] = file.IsDir()
	}

	return lis
}

func isBinary(filePath string) bool {
	// Code from:
	//		https://cs.opensource.google/go/x/tools/+refs/tags/v0.24:godoc/util/util.go:l=69
	file, err := os.Open(filePath)
	if err != nil {
		return true
	}
	defer file.Close()

	var buf [1024]byte
	n, err := file.Read(buf[0:])
	if err != nil {
		return true
	}

	return !isText(buf[0:n])
}

func isText(buf []byte) bool {
	// Code from:
	//		https://cs.opensource.google/go/x/tools/+refs/tags/v0.24:godoc/util/util.go:l=40
	const bMax = 1024
	if len(buf) > bMax {
		buf = buf[0:bMax]
	}

	for i, c := range string(buf) {
		if i+utf8.UTFMax > len(buf) {
			break
		}

		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' {
			return false
		}
	}
	return true
}
