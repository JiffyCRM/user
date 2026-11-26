package migrate

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"

	"github.com/JiffyCRM/user/migrations"
)

var (
	upFilePattern   = regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)
	downFilePattern = regexp.MustCompile(`^(\d+)_.*\.down\.sql$`)
)

type fileSet struct {
	up       map[int64]string
	down     map[int64]string
	versions []int64
}

func loadFiles() (*fileSet, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	files := &fileSet{
		up:   make(map[int64]string),
		down: make(map[int64]string),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if match := upFilePattern.FindStringSubmatch(name); match != nil {
			version, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse migration version %q: %w", name, err)
			}
			files.up[version] = name
			continue
		}
		if match := downFilePattern.FindStringSubmatch(name); match != nil {
			version, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parse migration version %q: %w", name, err)
			}
			files.down[version] = name
		}
	}

	if len(files.up) == 0 {
		return nil, errors.New("no up migrations found")
	}

	for version := range files.up {
		files.versions = append(files.versions, version)
	}
	sort.Slice(files.versions, func(i, j int) bool {
		return files.versions[i] < files.versions[j]
	})

	return files, nil
}