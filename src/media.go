package src

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func Media(w http.ResponseWriter, r *http.Request) {
	// Remove the first part
	url := strings.Split(r.URL.Path, "/")[2:]

	for _, dir := range config.Directories {
		if filepath.Base(dir) == url[0] {
			//Now open the file and serve it
			file := fmt.Sprintf("%s/%s", filepath.Dir(dir), strings.Join(url, "/"))
			fp, err := os.ReadFile(file)
			if err != nil {
				log.Println(err)
			}
			w.Write(fp)
		}
	}
}
