package slashresolve

import (
	"embed"
	"io/fs"
	"path"
)

//go:embed bundleddata
var bundledData embed.FS

func readBundledText(rel string) (string, error) {
	b, err := fs.ReadFile(bundledData, path.Join("bundleddata", rel))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
