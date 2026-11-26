package sqlquery

import (
	"fmt"
	"io/fs"
	"strings"
)

const marker = "-- name:"

type Set struct {
	queries map[string]string
}

func Load(fsys fs.FS, patterns ...string) (*Set, error) {
	set := &Set{queries: make(map[string]string)}
	for _, pattern := range patterns {
		matches, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no SQL files matched %q", pattern)
		}
		for _, name := range matches {
			data, err := fs.ReadFile(fsys, name)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", name, err)
			}
			if err := set.addFile(name, string(data)); err != nil {
				return nil, err
			}
		}
	}
	return set, nil
}

func (s *Set) Has(name string) bool {
	_, ok := s.queries[name]
	return ok
}

func (s *Set) MustGet(name string) string {
	query, ok := s.queries[name]
	if !ok {
		panic(fmt.Sprintf("sqlquery: missing query %q", name))
	}
	return query
}

func (s *Set) addFile(filename, contents string) error {
	var (
		currentName string
		body        strings.Builder
	)

	flush := func() error {
		if currentName == "" {
			return nil
		}
		query := strings.TrimSpace(body.String())
		if query == "" {
			return fmt.Errorf("%s: query %q is empty", filename, currentName)
		}
		if _, exists := s.queries[currentName]; exists {
			return fmt.Errorf("%s: duplicate query %q", filename, currentName)
		}
		s.queries[currentName] = query
		return nil
	}

	for _, line := range strings.Split(contents, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, marker) {
			if err := flush(); err != nil {
				return err
			}
			currentName = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			if currentName == "" {
				return fmt.Errorf("%s: empty query name", filename)
			}
			body.Reset()
			continue
		}
		if currentName != "" {
			body.WriteString(line)
			body.WriteString("\n")
			continue
		}
		if trimmed != "" {
			return fmt.Errorf("%s: SQL before the first %s marker", filename, marker)
		}
	}

	return flush()
}