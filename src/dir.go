package src

import (
	"bytes"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/gomarkdown/markdown"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type DirSt struct {
	Title   string
	Tag     string
	Content string
	IsMd    bool
	MdStr   string
	Navbar  string
	Entries int
	Files   int
	Dirs    int
}

type err_re struct {
	status int
	reason string
}

func renderDirectory(w http.ResponseWriter, r *http.Request, dirPath, cwd, relativePath string) {
	// Read the templates
	dirTmplData, err := templates.ReadFile("dir.tmpl")
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Can't read the directory template!")
		return
	}

	headerTmplData, err := templates.ReadFile("header.tmpl")
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Can't read the header template!")
		return
	}

	// Parse both templates together
	tmpl, err := template.New("dir").Parse(string(dirTmplData))
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error parsing the directory template")
		return
	}

	_, err = tmpl.New("header").Parse(string(headerTmplData))
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error parsing the header template!")
		return
	}

	dirIcon := `
<svg viewBox="0 0 16 16" class="svg octicon-file-directory-fill" aria-hidden="true" fill="#4793cc" width="16" height="16"><path d="M1.75 1A1.75 1.75 0 0 0 0 2.75v10.5C0 14.216.784 15 1.75 15h12.5A1.75 1.75 0 0 0 16 13.25v-8.5A1.75 1.75 0 0 0 14.25 3H7.5a.25.25 0 0 1-.2-.1l-.9-1.2C6.07 1.26 5.55 1 5 1z"></path></svg>
`
	fileIcon := `
<svg viewBox="0 0 16 16" class="svg octicon-file" aria-hidden="true" fill="#4793cc" width="16" height="16"><path d="M2 1.75C2 .784 2.784 0 3.75 0h6.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 13.25 16h-9.5A1.75 1.75 0 0 1 2 14.25Zm1.75-.25a.25.25 0 0 0-.25.25v12.5c0 .138.112.25.25.25h9.5a.25.25 0 0 0 .25-.25V6h-2.75A1.75 1.75 0 0 1 9 4.25V1.5Zm6.75.062V4.25c0 .138.112.25.25.25h2.688l-.011-.013-2.914-2.914z"></path></svg>
`

	type dispSt struct {
		path string
		name string
		size string
		mod  string
	}

	var filesIn []dispSt
	var dirsIn []dispSt
	var mdstr []byte
	var dmd bool
	// Render the header and directory listing
	files := cFilesWithDetails(dirPath)
	var dirs, fis int
	for _, file := range files {
		if strings.HasPrefix(strings.ToLower(file.Name), "readme") {
			dmd = true
			mdstr, err = os.ReadFile(filepath.Join(dirPath, file.Name))
			if err != nil {
				log.Println(err)
				InternalError(w, r, "Error reading README file!")
				return
			}
		}

		if file.IsDir {
			dirs++
			dirsIn = append(dirsIn, dispSt{
				path: filepath.Join(relativePath, file.Name),
				name: file.Name,
				size: fmt.Sprintf("(%s)", config.Name),
				mod:  humanize.Time(file.ModTime),
			})
		} else {
			fis++
			filesIn = append(filesIn, dispSt{
				path: filepath.Join(relativePath, file.Name),
				name: file.Name,
				size: fmt.Sprintf("(Approx. %s)", humanize.Bytes(uint64(file.Size))),
				mod:  humanize.Time(file.ModTime),
			})

		}
	}

	// Now start constructing the dir display.
	// This means it may take slightly longer since its now 2x the previous
	// design.
	var contentBuf strings.Builder
	// Start with dirs (as normal people do!)
	for _, dir := range dirsIn {
		contentBuf.WriteString(fmt.Sprintf(`
<div class="f-entry">
<p><span>%s<a href="/code/%s">%s</a></span> <span>%s</span>  <span>%s</span></p>
</div>
`, dirIcon, dir.path, trimName(dir.name, 70), dir.size, dir.mod))
	}

	for _, fi := range filesIn {
		contentBuf.WriteString(fmt.Sprintf(`
<div class="f-entry">
<p><span>%s<a href="/code/%s">%s</a></span> <span>%s</span>  <span>%s</span></p>
</div>
`, fileIcon, fi.path, trimName(fi.name, 70), fi.size, fi.mod))
	}

	if dmd {
		mdstr = append([]byte(`<div class="markdown_disp">`), markdown.ToHTML(mdstr, nil, nil)...)
		mdstr = append(mdstr, []byte(`</div>`)...)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "dir", DirSt{
		Title:   relativePath,
		Tag:     config.Tag,
		Entries: len(files),
		Files:   fis,
		Dirs:    dirs,
		Navbar:  navbar,
		Content: contentBuf.String(),
		IsMd:    dmd,
		MdStr:   string(mdstr),
	})
	if err != nil {
		log.Println(err)
		InternalError(w, r, "Error executing the template!")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(buf.Bytes())
}
