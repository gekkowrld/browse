package search

import (
	"bufio"
	"bytes"
	"codeberg.org/gekkowrld/browse/src"
	disp "codeberg.org/gekkowrld/browse/src/disp"
	"embed"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
)

const perPage int = 10

var templates embed.FS = src.Templates

type SearchData struct {
	For         string
	Results     string
	Time        string
	IsRes       bool
	ResLen      int
	Pages       int
	CurrentPage int
	HFirst      string
	HPrev       string
	HPage       string
	HNext       string
	HLast       string
}

type QueryRes struct {
	Filename    string
	Mod         time.Time
	Query       string
	URL         string
	LineContent string
	LineNumber  uint64
}

func Search(w http.ResponseWriter, r *http.Request) {
	searchTmpl, err := templates.ReadFile("templates/search.tmpl")
	if err != nil {
		log.Println(err)
		disp.NotFound(w, r, "search template not found!")
		return
	}
	headerTmpl, err := templates.ReadFile("templates/header.tmpl")
	if err != nil {
		log.Println(err)
		disp.NotFound(w, r, "header template not found!")
		return
	}

	tmpl := template.Must(template.New("search").Parse(string(searchTmpl)))
	_, err = tmpl.New("header").Parse(string(headerTmpl))
	if err != nil {
		log.Println(err)
		disp.InternalError(w, r, "Template parsing error")
		return
	}

	query := r.URL.Query().Get("query")
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	results := getQuery(query)

	totalResults := len(results)
	totalPages := (totalResults + perPage - 1) / perPage
	if page > totalPages {
		page = totalPages
	}

	startIndex := (page - 1) * perPage
	if startIndex < 0 {
		startIndex = 0
	}
	endIndex := startIndex + perPage
	if endIndex > totalResults {
		endIndex = totalResults
	}

	data := SearchData{
		For:         query,
		Results:     displayPage(results[startIndex:endIndex]),
		Time:        time.Since(time.Now()).String(),
		CurrentPage: page,
		Pages:       totalPages,
		IsRes:       totalResults > 0,
		ResLen:      totalResults,
	}

	// Generate pagination links
	data.HFirst = buildPaginationURL(query, 1)
	data.HPrev = buildPaginationURL(query, max(page-1, 1))
	data.HPage = buildPaginationURL(query, page)
	data.HNext = buildPaginationURL(query, min(page+1, totalPages))
	data.HLast = buildPaginationURL(query, totalPages)

	if err := tmpl.Execute(w, data); err != nil {
		disp.InternalError(w, r, err.Error())
		return
	}
}

func buildPaginationURL(query string, page int) string {
	return fmt.Sprintf("/search?query=%s&page=%d", url.QueryEscape(query), page)
}

func displayPage(results []QueryRes) string {
	var resultsHTML strings.Builder
	for _, res := range results {
		htmlContent := html.EscapeString(res.LineContent)
		replacedStr := strings.ReplaceAll(src.TrimText(htmlContent, 101), res.Query, fmt.Sprintf(`<span class="replace_str">%s</span>`, res.Query))
		resultsHTML.WriteString(fmt.Sprintf(`
<div class="search-result">
<a href="%s" title="%s">%s</a> <span>%s:%d</span>
</div>
`, res.URL, htmlContent, replacedStr, src.TrimText(res.Filename, 32, true), res.LineNumber))
	}

	return resultsHTML.String()
}

func getQuery(query string) []QueryRes {
	if strings.TrimSpace(query) == "" {
		return []QueryRes{}
	}
	dirs := disp.SetConfig().Directories
	if strings.Contains(query, "dir") {
		re := regexp.MustCompile(`(?m)dir:(\w+)`)
		matches := re.FindStringSubmatch(query)
		if len(matches) > 1 {
			match := matches[1]
			dirs = filterDirsByMatch(dirs, match)
			query = re.ReplaceAllString(query, "")
			query = strings.TrimSpace(query)
		}
	}

	ext_ := regexp.MustCompile(`(?m)ext:([\w.]+)`)
	var results []QueryRes
	var wg sync.WaitGroup
	for _, dir := range dirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			var ext_s string
			if strings.Contains(query, "ext") {
				matches := ext_.FindStringSubmatch(query)
				if len(matches) > 1 {
					ext_s = matches[1]
					query = ext_.ReplaceAllString(query, "")
					query = strings.TrimSpace(query)
				}
			}
			res := walkDir(ext_s, d, query)
			if res != nil {
				mutex.Lock()
				results = append(results, res...)
				mutex.Unlock()
			}
		}(dir)
	}
	wg.Wait()
	return results
}

func filterDirsByMatch(dirs []string, match string) []string {
	var filtered []string
	for _, dir := range dirs {
		if strings.Contains(filepath.Base(dir), match) {
			filtered = append(filtered, dir)
		}
	}
	return filtered
}

var mutex sync.Mutex

func walkDir(ext, dir, query string) []QueryRes {
	var results []QueryRes
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.IsDir() {
			relPath, _ := filepath.Rel(dir, path)
			relPath = filepath.Join(filepath.Base(dir), relPath)

			fext := filepath.Ext(relPath)
			//	fmt.Println(fext, "sys--user", ext)
			if len(ext) > 1 && !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}

			// Compare file extension
			if len(ext) > 1 && fext != ext {
				return nil
			}
			res, err := searchFile(relPath, path, query)
			if err == nil && len(res) > 0 {
				mutex.Lock()
				results = append(results, res...)
				mutex.Unlock()
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return results
}

func searchFile(relpath, file, query string) ([]QueryRes, error) {
	if isBinary := src.IsBinary(file); isBinary {
		return nil, fmt.Errorf("file is a binary")
	}

	fp, err := os.Open(file)
	if err != nil {
		return []QueryRes{}, err
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	var results []QueryRes
	var lineNumber uint64
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if bytes.Contains(line, []byte(query)) {
			results = append(results, QueryRes{
				Filename:    file,
				Query:       query,
				LineNumber:  lineNumber,
				LineContent: string(line),
				URL:         fmt.Sprintf("/code/%s#L%d", relpath, lineNumber),
			})
		}
	}
	return results, nil
}
