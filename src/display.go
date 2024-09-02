package src

import (
	"bytes"
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

//go:embed *tmpl
var templates embed.FS

type IndexSt struct {
	Title   string
	Content string
}

type FileDetail struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime time.Time
}

func Code(w http.ResponseWriter, r *http.Request) {
	var html_c []byte

	navbar = navbarSec(r)
	// Handle URL path
	urlPath := r.URL.Path
	if urlPath == "/code/" {
		html_c = codeIndex()
	} else {
		cfgPath, _ := expandPath("~/.config/browse/config.ini")
		conf, err := LoadConfig(cfgPath)
		if err != nil {
			log.Println(err)
			w.Write(NotFound())
			return
		}

		var cwd string
		for _, dir := range conf.Directories {
			// Get the 'first' part of the path string
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) >= 3 {
				fparts := parts[2]
				if fparts == filepath.Base(dir) {
					cwd = dir
				}
			}
		}

		html_c = otherUrls(filepath.Dir(cwd), urlPath)
	}

	if html_c == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	w.Write(html_c)
}

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

func otherUrls(cwd string, urlPath string) []byte {
	// Trim the "/code/" prefix from the URL path to get the relative path
	filePath := strings.TrimPrefix(urlPath, "/code/")
	fullPath := filepath.Join(cwd, filePath)

	// Check if the path exists and is accessible
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NotFound()
		}
		log.Println(err)
		return NotFound()
	}

	if fileInfo.IsDir() {
		return renderDirectory(fullPath, cwd, filePath)
	}

	return renderFile(fullPath, filePath)
}

func codeIndex() []byte {
	data, err := templates.ReadFile("index.tmpl")
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	tmpl, err := template.New("index").Parse(string(data))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	cfgPath, _ := expandPath("~/.config/browse/config.ini")
	conf, err := LoadConfig(cfgPath)
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	var fid strings.Builder
	for _, file := range conf.Directories {
		fn := filepath.Base(file)
		fid.WriteString(fmt.Sprintf(`<p><a href="/code/%s">%s</a> <span>%s</span></p>`, fn, fn, file))
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, IndexSt{Title: "Code Directories", Content: fid.String()})
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	return buf.Bytes()
}
