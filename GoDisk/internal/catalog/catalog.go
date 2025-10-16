package catalog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type catalog struct {
	Version int      `json:"version"`
	Disks   []string `json:"disks"`
}

var (
	mu        sync.Mutex
	cachePath string
)

func CurrentPath() string {
	p, _ := path()
	return p
}

func path() (string, error) {
	if cachePath != "" {
		return cachePath, nil
	}

	if p := os.Getenv("GODISK_CATALOG"); strings.TrimSpace(p) != "" {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", err
		}
		cachePath = p
		return cachePath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".godisk")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	cachePath = filepath.Join(dir, "catalog.json")
	return cachePath, nil
}

func load() (catalog, error) {
	var c catalog
	p, err := path()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return catalog{Version: 1, Disks: []string{}}, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.Disks == nil {
		c.Disks = []string{}
	}
	return c, nil
}

func save(c catalog) error {
	p, err := path()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func Add(diskPath string) error {
	mu.Lock()
	defer mu.Unlock()
	c, err := load()
	if err != nil {
		return err
	}
	diskPath = filepath.Clean(diskPath)
	for _, x := range c.Disks {
		if x == diskPath {
			return nil
		}
	}
	c.Disks = append(c.Disks, diskPath)
	sort.Strings(c.Disks)
	return save(c)
}

func Remove(diskPath string) error {
	mu.Lock()
	defer mu.Unlock()
	c, err := load()
	if err != nil {
		return err
	}
	diskPath = filepath.Clean(diskPath)
	out := c.Disks[:0]
	for _, x := range c.Disks {
		if x != diskPath {
			out = append(out, x)
		}
	}
	if len(out) == len(c.Disks) {
		return nil
	}
	c.Disks = out
	return save(c)
}

// All devuelve una copia de las rutas registradas.
func All() ([]string, error) {
	mu.Lock()
	defer mu.Unlock()
	c, err := load()
	if err != nil {
		return nil, err
	}
	cp := make([]string, len(c.Disks))
	copy(cp, c.Disks)
	return cp, nil
}
