package gits

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type DiskFS struct {
	root string
}

func NewDiskFS(root string) (FS, error) {
	disk := &DiskFS{}

	if err := disk.SetRoot(root); err != nil {
		return nil, err
	}

	return disk, nil
}

func (d *DiskFS) SetRoot(path string) error {
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

func (d *DiskFS) Abs(path string) string {
	// Remove null bytes and control chars
	path = strings.Map(func(r rune) rune {
		if r < 32 || r == '\x00' {
			return -1 // drop it
		}
		return r
	}, path)

	// Normalize
	clean := filepath.Clean(path)

	if !filepath.IsAbs(clean) {
		clean = filepath.Join(d.root, clean)
	}
	return clean
}

func (d *DiskFS) ReadFile(path string) ([]byte, error) {
	filepath := d.Abs(path)
	return os.ReadFile(filepath)
}

func (d *DiskFS) WriteFile(path string, data []byte) error {
	full := d.Abs(path)

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}

	return os.WriteFile(full, data, 0644)
}

func (d *DiskFS) ReadBatch(paths []string) (map[string][]byte, error) {
	result := make(map[string][]byte)

	for _, p := range paths {
		data, err := d.ReadFile(p)
		if err != nil {
			return nil, err
		}
		result[p] = data
	}

	return result, nil
}

func (d *DiskFS) WriteBatch(data map[string][]byte) error {
	for p, content := range data {
		if err := d.WriteFile(p, content); err != nil {
			return err
		}
	}

	return nil
}

func (d *DiskFS) Scan(path string, level int) (map[string][]int, error) {
	// Trim left /
	path = strings.TrimPrefix(path, "/")
	// Trim right /
	path = strings.TrimSuffix(path, "/")

	result := make(map[string][]int)
	base := d.Abs(path)

	err := filepath.WalkDir(base, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if p == base {
			return nil // skip root itself
		}

		rel, _ := filepath.Rel(base, p)

		// Apply depth restriction if level >= 0
		if level >= 0 {
			depth := len(strings.Split(rel, string(os.PathSeparator))) - 1
			if depth > level {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}

			// Prepend the passed `path` to the relative path
			key := filepath.Join(path, rel)
			result[key] = []int{1, int(info.Size())} // TYPE=1 for file
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (d *DiskFS) Stat(path string) []int {
	full := d.Abs(path)
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

func (d *DiskFS) MkdirAll(path string) error {
	full := d.Abs(path)
	return os.MkdirAll(full, 0755)
}

func (d *DiskFS) Cd(path string) error {
	return d.SetRoot(d.Abs(path))
}
