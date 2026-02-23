package resolver

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LearningResolver resolves learning IDs to filesystem paths.
type LearningResolver interface {
	Resolve(id string) (path string, err error)
}

// FileResolver resolves learnings by searching the filesystem.
type FileResolver struct {
	Root string // project root (where .agents/ lives)
}

// NewFileResolver creates a FileResolver rooted at the given directory.
func NewFileResolver(root string) *FileResolver {
	return &FileResolver{Root: root}
}

// extensions lists the file extensions to probe when searching for learnings.
var extensions = []string{".jsonl", ".md", ".json"}

// subdirs lists the subdirectories under .agents that contain learnings.
var subdirs = []string{"learnings", "patterns"}

// Resolve locates a learning file by ID, searching .agents/learnings/ and
// .agents/patterns/ with extension probing, direct path, glob, and
// frontmatter ID scanning. It walks up parent directories to the rig root.
func (r *FileResolver) Resolve(id string) (string, error) {
	// Normalize: strip pend- prefix for pool IDs
	normalized := id
	if strings.HasPrefix(normalized, "pend-") {
		normalized = strings.TrimPrefix(normalized, "pend-")
	}

	// If it looks like an absolute path, try converting to relative
	if filepath.IsAbs(normalized) {
		rel, err := filepath.Rel(r.Root, normalized)
		if err == nil && !strings.HasPrefix(rel, "..") {
			// It's within our root — try it as a direct file
			if _, statErr := os.Stat(normalized); statErr == nil {
				return normalized, nil
			}
			// Also try the basename as an ID
			normalized = filepath.Base(normalized)
			// Strip extension for ID-based lookup
			ext := filepath.Ext(normalized)
			for _, e := range extensions {
				if ext == e {
					normalized = strings.TrimSuffix(normalized, ext)
					break
				}
			}
		}
	}

	baseDirs := buildAgentsDirs(r.Root)

	if p, err := searchDirs(baseDirs, normalized); err != nil || p != "" {
		return p, err
	}

	// Walk up to rig root looking for .agents/learnings and .agents/patterns
	dir := r.Root
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent

		candidates := buildAgentsDirs(dir)
		// Skip directories already searched at root level
		var novel []string
		for _, c := range candidates {
			if !inSlice(c, baseDirs) {
				novel = append(novel, c)
			}
		}
		for _, c := range novel {
			if p := probeWithExtensions(c, normalized); p != "" {
				return p, nil
			}
		}
	}

	return "", fmt.Errorf("learning not found: %s", id)
}

// buildAgentsDirs returns .agents/<subdir> paths for a given root.
func buildAgentsDirs(root string) []string {
	dirs := make([]string, 0, len(subdirs))
	for _, sub := range subdirs {
		dirs = append(dirs, filepath.Join(root, ".agents", sub))
	}
	return dirs
}

// searchDirs tries extension-probing, direct-probing, glob-probing,
// and frontmatter ID scanning across dirs.
func searchDirs(dirs []string, id string) (string, error) {
	for _, d := range dirs {
		if p := probeWithExtensions(d, id); p != "" {
			return p, nil
		}
	}
	for _, d := range dirs {
		if p := probeDirect(d, id); p != "" {
			return p, nil
		}
	}
	for _, d := range dirs {
		p, err := probeGlob(d, id)
		if err != nil {
			return "", err
		}
		if p != "" {
			return p, nil
		}
	}
	for _, d := range dirs {
		if p := probeFrontmatterID(d, id); p != "" {
			return p, nil
		}
	}
	return "", nil
}

// probeWithExtensions checks for id + each extension inside dirPath.
func probeWithExtensions(dirPath, id string) string {
	for _, ext := range extensions {
		path := filepath.Join(dirPath, id+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// probeDirect checks if id exists as-is inside dirPath (for IDs that already include an extension).
func probeDirect(dirPath, id string) string {
	path := filepath.Join(dirPath, id)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// probeGlob searches for files whose names contain id inside dirPath.
func probeGlob(dirPath, id string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dirPath, "*"+id+"*"))
	if err != nil {
		return "", err
	}
	if len(files) > 0 {
		return files[0], nil
	}
	return "", nil
}

// probeFrontmatterID scans .md files in dirPath for a frontmatter "id" field matching id.
func probeFrontmatterID(dirPath, id string) string {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dirPath, e.Name())
		fmID, err := readFrontmatterField(path, "id")
		if err != nil {
			continue
		}
		if fmID == id {
			return path
		}
	}
	return ""
}

// readFrontmatterField extracts a single field value from YAML frontmatter.
func readFrontmatterField(path, field string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	dashCount := 0
	prefix := field + ":"

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			dashCount++
			if dashCount == 1 {
				inFrontmatter = true
				continue
			}
			if dashCount == 2 {
				break
			}
		}

		if inFrontmatter && strings.HasPrefix(trimmed, prefix) {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			val = strings.Trim(val, "\"'")
			return val, nil
		}
	}

	return "", scanner.Err()
}

// inSlice returns true if needle is present in the slice.
func inSlice(needle string, slice []string) bool {
	for _, s := range slice {
		if needle == s {
			return true
		}
	}
	return false
}
