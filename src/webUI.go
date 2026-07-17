package src

import (
	"embed"
	"io/fs"
)

//go:embed all:html
var htmlFS embed.FS

var webUI = make(map[string]interface{})

func loadHTMLMap() {
	fs.WalkDir(htmlFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := htmlFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		webUI[path] = string(data)
		return nil
	})
}

func GetHTMLString(base string) string {
	return base
}
