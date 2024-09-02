package src

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/dustin/go-humanize"
	"github.com/go-ini/ini"
	"github.com/gomarkdown/markdown"
	ghtml "html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"
)

const maxReadSize = 4096 // Number of bytes to read for checking
var navbar string

//go:embed *tmpl
var templates embed.FS

type IndexSt struct {
	Title   string
	Content string
}

type DirSt struct {
	Title   string
	Content string
	IsMd    bool
	MdStr   string
	Navbar  string
	Entries int
	Files   int
	Dirs    int
}

type FileSt struct {
	Title   string
	Content string
	Navbar  string
}

type Config struct {
	Directories   []string `ini:"dirs"`
	PreferredName string   `ini:"preferred_name"`
}

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

func LoadConfig(filename string) (*Config, error) {
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = cfg.Section("directories").MapTo(&config)
	if err != nil {
		return nil, err
	}

	err = cfg.Section("settings").MapTo(&config)
	if err != nil {
		return nil, err
	}

	// Check for '*' first
	var cdirs []string
	for _, dir := range config.Directories {
		if strings.HasSuffix(dir, "*") {
			expandedPath, err := expandPath(filepath.Dir(dir))
			if err != nil {
				return nil, err
			}

			dirs, err := os.ReadDir(expandedPath)
			if err != nil {
				return nil, err
			}

			for _, dir_s := range dirs {
				cdirs = append(cdirs, fmt.Sprintf("%s/%s", filepath.Dir(dir), dir_s.Name()))
			}
		} else {
			cdirs = append(cdirs, dir)
		}
	}

	config.Directories = uniqueSortedEntries(cdirs)

	// Expand and resolve directories
	var expandedDirs []string
	for _, dir := range config.Directories {
		expandedPath, err := expandPath(dir)
		if err != nil {
			return nil, err
		}
		resolvedPath, err := resolvePath(expandedPath)
		if err != nil {
			return nil, err
		}
		expandedDirs = append(expandedDirs, resolvedPath)
	}
	config.Directories = expandedDirs

	return &config, nil
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

func isBinary(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read a portion of the file
	buffer := make([]byte, maxReadSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Check for non-printable characters
	nonPrintableCount := 0
	for i := 0; i < n; i++ {
		if !unicode.IsPrint(rune(buffer[i])) && buffer[i] != '\n' && buffer[i] != '\r' && buffer[i] != '\t' {
			nonPrintableCount++
		}
	}

	// If a significant portion of non-printable characters, it's likely binary
	return nonPrintableCount > maxReadSize/10, nil
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

func renderFile(filename, relpath string) []byte {
	var fileContent []byte
	isb, err := isBinary(filename)
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	if isb {
		if !isViewableInBrowser(filename) {
			fileContent = []byte("The file is a binary file, can't display it")
		} else {
			fileContent, err = os.ReadFile(filename)
			if err != nil {
				log.Println(err)
				return NotFound()
			}
		}
	}

	if !isb {
		fileContent, err = os.ReadFile(filename)
		if err != nil {
			log.Println(err)
			return NotFound()
		}
	}

	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(string(fileContent))
		if lexer == nil {
			lexer = lexers.Fallback
		}
	}

	iterator, err := lexer.Tokenise(nil, string(fileContent))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	style := styles.Get("catppuccin-mocha")
	if style == nil {
		style = styles.Fallback
	}

	// Use chroma to highlight the code
	formatter := html.New(html.Standalone(true), html.WithClasses(true), html.WithLineNumbers(true), html.WithLinkableLineNumbers(true, "L"), html.WrapLongLines(true))
	var highlighted bytes.Buffer
	err = formatter.Format(&highlighted, style, iterator)
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	// Read the templates
	fileTmplData, err := templates.ReadFile("file.tmpl")
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	// Parse both templates together
	tmpl, err := template.New("file").Parse(string(fileTmplData))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "file", FileSt{Title: relpath, Navbar: navbar, Content: highlighted.String()})
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	return buf.Bytes()

}

func renderDirectory(dirPath, cwd, relativePath string) []byte {
	// Read the templates
	dirTmplData, err := templates.ReadFile("dir.tmpl")
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	// Parse both templates together
	tmpl, err := template.New("dir").Parse(string(dirTmplData))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	dirIcon := `
<svg viewBox="0 0 16 16" class="svg octicon-file-directory-fill" aria-hidden="true" fill="#4793cc" width="16" height="16"><path d="M1.75 1A1.75 1.75 0 0 0 0 2.75v10.5C0 14.216.784 15 1.75 15h12.5A1.75 1.75 0 0 0 16 13.25v-8.5A1.75 1.75 0 0 0 14.25 3H7.5a.25.25 0 0 1-.2-.1l-.9-1.2C6.07 1.26 5.55 1 5 1z"></path></svg>
`
	fileIcon := `
<svg viewBox="0 0 16 16" class="svg octicon-file" aria-hidden="true" fill="#4793cc" width="16" height="16"><path d="M2 1.75C2 .784 2.784 0 3.75 0h6.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 13.25 16h-9.5A1.75 1.75 0 0 1 2 14.25Zm1.75-.25a.25.25 0 0 0-.25.25v12.5c0 .138.112.25.25.25h9.5a.25.25 0 0 0 .25-.25V6h-2.75A1.75 1.75 0 0 1 9 4.25V1.5Zm6.75.062V4.25c0 .138.112.25.25.25h2.688l-.011-.013-2.914-2.914z"></path></svg>
`

	var mdstr []byte
	var dmd bool
	// Render the header and directory listing
	var contentBuf strings.Builder
	files := cFilesWithDetails(dirPath)
	var dirs, fis int
	for _, file := range files {
		var fsize string
		var icon string

		if strings.HasPrefix(strings.ToLower(file.Name), "readme") {
			dmd = true
			mdstr, err = os.ReadFile(filepath.Join(dirPath, file.Name))
			if err != nil {
				log.Println(err)
			}
		}

		if !file.IsDir {
			fsize = fmt.Sprintf("(Approx. %s)", humanize.Bytes(uint64(file.Size)))
			icon = fileIcon
			fis++
		} else {
			icon = dirIcon
			dirs++
		}
		contentBuf.WriteString(fmt.Sprintf(`
<div class="f-entry">
<p><span>%s<a href="/code/%s">%s</a></span> <span>%s</span>  <span>%s</span></p>
</div>
`, icon, filepath.Join(relativePath, file.Name), file.Name, fsize, humanize.Time(file.ModTime)))
	}

	if dmd {
		mdstr = append([]byte(`<div class="markdown_disp">`), markdown.ToHTML(mdstr, nil, nil)...)
		mdstr = append(mdstr, []byte(`</div>`)...)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "dir", DirSt{Title: relativePath, Entries: len(files), Files: fis, Dirs: dirs, Navbar: navbar, Content: contentBuf.String(), IsMd: dmd, MdStr: string(mdstr)})
	if err != nil {
		log.Println(err)
		return NotFound()
	}

	return buf.Bytes()
}

type FileDetail struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime time.Time
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

func Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.Write(NotFound())
		return
	}

	data, err := templates.ReadFile("home.tmpl")
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
		return
	}

	tmpl, err := template.New("index").Parse(string(data))
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
		return
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
		return
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	if err != nil {
		log.Println(err)
		w.Write(NotFound())
		return
	}

	w.Write(buf.Bytes())
}

func NotFound() []byte {
	return []byte("Not Found!")
}
