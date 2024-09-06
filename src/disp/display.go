package disp

import (
	"bytes"
	"codeberg.org/gekkowrld/browse/src"
	"embed"
	"fmt"
	ghtml "html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var navbar string
var config Config

var templates embed.FS = src.Templates

type IndexSt struct {
	Title   string
	Tag     string
	Content string
}

type FileDetail struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime time.Time
}

// Code handler
func Code(w http.ResponseWriter, r *http.Request) {
	// Set the configuration
	config = *SetConfig()

	navbar = navbarSec(r)
	// Handle URL path
	urlPath := r.URL.Path
	if urlPath == "/code/" {
		codeIndex(w, r)
	} else {
		var cwd string
		for _, dir := range config.Directories {
			// Get the 'first' part of the path string
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) >= 3 {
				fparts := parts[2]
				if fparts == filepath.Base(dir) {
					cwd = dir
				}
			}
		}

		otherUrls(w, r, filepath.Dir(cwd))
	}
}

// Build the navigation bar
func navbarSec(r *http.Request) string {
	urlPath := r.URL.Path

	// Split the URL path into segments
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")

	var breadcrumb strings.Builder
	var currentPath strings.Builder

	// Start with a home link
	breadcrumb.WriteString(`<a href="/">Home</a>`)

	// Build the breadcrumb trail
	for i, part := range parts {
		// Skip empty parts
		if part == "" {
			continue
		}

		currentPath.WriteString("/")
		currentPath.WriteString(part)

		// If not the last part, make it a clickable link
		if i < len(parts)-1 {
			breadcrumb.WriteString(fmt.Sprintf(` > <a href="%s">%s</a>`, currentPath.String(), ghtml.EscapeString(part)))
		} else {
			// For the last part, display it as plain text
			breadcrumb.WriteString(fmt.Sprintf(` > %s`, ghtml.EscapeString(part)))
		}
	}

	return breadcrumb.String()
}

// Handle requests for other URLs
func otherUrls(w http.ResponseWriter, r *http.Request, cwd string) {
	urlPath := r.URL.Path
	// Trim the "/code/" prefix from the URL path to get the relative path
	filePath := strings.TrimPrefix(urlPath, "/code/")
	fullPath := filepath.Join(cwd, filePath)

	// Check if the path exists and is accessible
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			NotFound(w, r, "The path was not found")
			return
		}
		log.Println(err)
		InternalError(w, r, "Error accessing the file or directory!")
		return
	}

	if fileInfo.IsDir() {
		renderDirectory(w, r, fullPath, cwd, filePath)
		return
	}

	renderFile(w, r, fullPath, filePath)
	return
}

// Generate the index page
func codeIndex(w http.ResponseWriter, r *http.Request) {
	data, err := templates.ReadFile("templates/index.tmpl")
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error reading the index template!")
		return
	}

	headerTmplData, err := templates.ReadFile("templates/header.tmpl")
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error reading the header template!")
		return
	}

	tmpl, err := template.New("index").Parse(string(data))
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error parsing the index template!")
		return
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error parsing the header template!")
		return
	}

	var fid strings.Builder
	for _, file := range config.Directories {
		fn := filepath.Base(file)
		fid.WriteString(fmt.Sprintf(`<p><a href="/code/%s">%s</a> <span>%s</span></p>`, fn, src.TrimText(fn, 15), src.TrimText(file, 30, true)))
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, IndexSt{Title: "Code Directories", Content: fid.String(), Tag: config.Tag})
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error executing the index template!")
		return
	}

	w.Write(buf.Bytes())
}
