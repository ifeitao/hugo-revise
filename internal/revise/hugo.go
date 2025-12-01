package revise

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ifeitao/hugo-revise/internal/fm"
)

// getPageURLFromHugo uses hugo list all to get the actual permalink
func getPageURLFromHugo(bundleDir string, frontMatter fm.FrontMatter) (string, error) {
	// Find Hugo project root
	projectRoot, err := findHugoRoot(bundleDir)
	if err != nil {
		return "", err
	}

	// Run hugo list all to get CSV output
	cmd := exec.Command("hugo", "list", "all")
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("hugo list all failed: %w", err)
	}

	// Parse CSV output
	reader := csv.NewReader(strings.NewReader(string(output)))
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("parse csv failed: %w", err)
	}

	// Find header indices
	if len(records) < 2 {
		return "", fmt.Errorf("no content found in hugo list all output")
	}

	header := records[0]
	pathIdx := -1
	permalinkIdx := -1
	for i, col := range header {
		if col == "path" {
			pathIdx = i
		} else if col == "permalink" {
			permalinkIdx = i
		}
	}

	if pathIdx == -1 || permalinkIdx == -1 {
		return "", fmt.Errorf("required columns not found in hugo list all output")
	}

	// Get relative path from content directory
	contentPath := filepath.Join(projectRoot, "content")
	relPath, err := filepath.Rel(contentPath, bundleDir)
	if err != nil {
		return "", err
	}

	// Normalize the bundle directory path to match hugo list all output
	// hugo list all shows index.md for bundles, or .md for single files
	targetPath := ""
	indexMd := filepath.Join(bundleDir, "index.md")
	if _, err := os.Stat(indexMd); err == nil {
		// It's a bundle, match against path/index.md
		targetPath = filepath.Join(relPath, "index.md")
	} else {
		// Single file .md
		targetPath = relPath + ".md"
	}

	// Search for matching path in records
	for _, record := range records[1:] {
		if len(record) <= pathIdx || len(record) <= permalinkIdx {
			continue
		}

		recordPath := record[pathIdx]
		// Normalize path separators for comparison
		recordPath = filepath.FromSlash(recordPath)

		if strings.Contains(recordPath, targetPath) || strings.HasSuffix(recordPath, targetPath) {
			permalink := record[permalinkIdx]
			// Extract path from full URL (remove baseURL)
			// e.g., https://yifeitao.com/entertainment-unlimited/ -> /entertainment-unlimited/
			if idx := strings.Index(permalink, "://"); idx != -1 {
				// Find first / after ://
				if slashIdx := strings.Index(permalink[idx+3:], "/"); slashIdx != -1 {
					return permalink[idx+3+slashIdx:], nil
				}
			}
			return permalink, nil
		}
	}

	return "", fmt.Errorf("page not found in hugo list all output")
}

func findHugoRoot(startPath string) (string, error) {
	dir := startPath
	for {
		// Check for Hugo config files
		for _, name := range []string{"hugo.toml", "hugo.yaml", "hugo.yml", "config.toml", "config.yaml", "config.yml"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("Hugo project root not found")
}
