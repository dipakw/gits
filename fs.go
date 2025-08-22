package gits

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type DiskFS struct {
	root string
	dir  string
}

func NewDiskFS(root string) (FS, error) {
	disk := &DiskFS{
		dir: "/",
	}

	if err := disk.setRoot(root); err != nil {
		return nil, err
	}

	return disk, nil
}

func (d *DiskFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(d.abs(path))
}

func (d *DiskFS) WriteFile(path string, data []byte) error {
	full := d.abs(path)

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}

	return os.WriteFile(full, data, 0644)
}

func (d *DiskFS) Scan(path string, include uint8, level int) (map[string][]int, error) {
	result := make(map[string][]int)

	if include == 0 {
		return result, nil
	}

	includeDirs := include&FS_TYPE_DIR != 0
	includeFiles := include&FS_TYPE_FILE != 0

	root := d.abs(path)
	prefix := path

	var walk func(curr string, depth int) error

	walk = func(curr string, depth int) error {
		entries, err := os.ReadDir(curr)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			fullPath := filepath.Join(curr, entry.Name())
			relPath, _ := filepath.Rel(root, fullPath)
			key := filepath.ToSlash(filepath.Join(prefix, relPath))

			if entry.IsDir() {
				if includeDirs {
					result[key] = []int{2, 0}
				}

				// if level == -1 => unlimited depth
				if level == -1 || depth < level {
					if err := walk(fullPath, depth+1); err != nil {
						return err
					}
				}
			} else if includeFiles {
				info, err := entry.Info()

				if err != nil {
					return err
				}

				result[key] = []int{1, int(info.Size())}
			}
		}

		return nil
	}

	if err := walk(root, 0); err != nil {
		return map[string][]int{}, err
	}

	return result, nil
}

func (d *DiskFS) Stat(path string) []int {
	full := d.abs(path)
	info, err := os.Stat(full)

	if errors.Is(err, os.ErrNotExist) {
		return []int{0, 0} // not found
	}

	if err != nil {
		// If there's an unexpected error, treat as not found
		return []int{0, 0}
	}

	if info.IsDir() {
		return []int{2, int(info.Size())}
	}

	return []int{1, int(info.Size())}
}

func (d *DiskFS) Mkdir(path string) error {
	return os.MkdirAll(d.abs(path), 0755)
}

func (d *DiskFS) Cd(path string) error {
	if strings.HasPrefix(path, "/") {
		d.dir = path
	} else {
		d.dir = filepath.Join(d.dir, path)
	}

	return nil
}

func (d *DiskFS) Pwd() string {
	return d.dir
}

// Helper functions.
func (d *DiskFS) setRoot(path string) error {
	absPath, err := filepath.Abs(path)

	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)

	if err != nil {
		return err
	}

	if !info.IsDir() {
		return errors.New("root path must be a directory")
	}

	d.root = absPath

	return nil
}

func (d *DiskFS) abs(path string) string {
	// Remove null bytes and control chars
	path = strings.Map(func(r rune) rune {
		if r < 32 || r == '\x00' {
			return -1 // drop it
		}
		return r
	}, path)

	// Normalize
	clean := filepath.Clean(path)

	if strings.HasPrefix(clean, "/") {
		clean = filepath.Join(d.root, clean)
	} else {
		clean = filepath.Join(d.root, d.dir, clean)
	}

	return clean
}
