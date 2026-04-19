package main

import (
	"fmt"
	"path/filepath"
)

func main() {
	paths := []string{"./file.md", "file.md", "/absolute/path", "~/home/path"}
	for _, path := range paths {
		fmt.Printf("path: %q, IsAbs: %v\n", path, filepath.IsAbs(path))
	}
}
