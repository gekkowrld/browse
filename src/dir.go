package src

import (
	"bytes"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/gomarkdown/markdown"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

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
