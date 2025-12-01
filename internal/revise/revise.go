package revise

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ifeitao/hugo-revise/internal/config"
	"github.com/ifeitao/hugo-revise/internal/fm"
)

type lastOp struct {
	Timestamp string   `json:"timestamp"`
	Changes   []change `json:"changes"`
}

type change struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Action string `json:"action"` // copy, move, write
}

func Run(cfg config.Config, pathPrefix string) error {
	if err := config.EnsureLogDir(); err != nil {
		return err
	}

	var sourceFile string
	var isBundle bool
	var bundleName string

	// Smart path detection
	if strings.HasSuffix(pathPrefix, ".md") {
		// Explicit .md file
		sourceFile = pathPrefix
		isBundle = false
		bundleName = strings.TrimSuffix(filepath.Base(pathPrefix), ".md")
	} else {
		// Could be either a bundle directory or a file name without extension
		// Try as bundle first (check for index.md)
		bundleIndexPath := filepath.Join(pathPrefix, "index.md")
		if _, err := os.Stat(bundleIndexPath); err == nil {
			// It's a bundle with index.md
			sourceFile = bundleIndexPath
			isBundle = true
			bundleName = filepath.Base(pathPrefix)
		} else {
			// Try adding .md extension
			mdPath := pathPrefix + ".md"
			if _, err := os.Stat(mdPath); err == nil {
				// It's a single .md file
				sourceFile = mdPath
				isBundle = false
				bundleName = filepath.Base(pathPrefix)
			} else {
				return fmt.Errorf("source not found: tried %s and %s", bundleIndexPath, mdPath)
			}
		}
	}

	// Verify source file exists (redundant check but clear error message)
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", sourceFile)
	}

	// Read source content
	b, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}
	parsed, err := fm.Parse(string(b))
	if err != nil {
		return err
	}

	// Create revisions directory (e.g., my-post.revisions/ or my-post-bundle.revisions/)
	baseDir := filepath.Dir(pathPrefix)
	revisionsDir := filepath.Join(baseDir, bundleName+".revisions")
	if err := os.MkdirAll(revisionsDir, 0o755); err != nil {
		return err
	}

	// Determine version labels
	baseDate := extractDocumentDate(parsed, cfg.Versioning.DateFormat)
	currentDate := time.Now().Format(cfg.Versioning.DateFormat)

	// Check if a revision for today already exists
	existingEntries, _ := os.ReadDir(revisionsDir)
	for _, e := range existingEntries {
		name := e.Name()
		var versionLabel string
		if isBundle {
			if e.IsDir() {
				versionLabel = name
			}
		} else {
			if !e.IsDir() && strings.HasSuffix(name, ".md") {
				versionLabel = strings.TrimSuffix(name, ".md")
			}
		}
		if versionLabel == currentDate {
			return fmt.Errorf("a revision for %s already exists. hugo-revise is designed for major revisions, not daily updates. Please use git for granular version control, or wait until a different day to create another revision", currentDate)
		}
	}

	// Archive the old content with its original date
	version := baseDate
	// Current version uses today's date
	newLatestLabel := currentDate

	// Create archived target
	var archivedFile string
	var archivedDir string
	if isBundle {
		archivedDir = filepath.Join(revisionsDir, version)
		if err := os.MkdirAll(archivedDir, 0o755); err != nil {
			return err
		}
		archivedFile = filepath.Join(archivedDir, "index.md")
	} else {
		archivedFile = filepath.Join(revisionsDir, version+".md")
	}

	// Prepare archived content
	archivedFM := parsed

	// Determine base URL
	var baseURL string
	if isBundle {
		baseURL = extractBaseURL(parsed, pathPrefix)
	} else {
		baseURL = extractBaseURL(parsed, filepath.Join(baseDir, bundleName))
	}

	archiveURL := fmt.Sprintf("%srevisions/%s/", baseURL, version)

	// Set fixed URL for archived version
	archivedFM, _ = fm.InjectKV(archivedFM, "url", archiveURL)
	archivedFM, _ = fm.InjectBuildOptions(archivedFM)

	// Build revisions_history: scan archived versions + current
	var versions []string
	entries, _ := os.ReadDir(revisionsDir)
	for _, e := range entries {
		name := e.Name()
		if isBundle {
			if e.IsDir() {
				versions = append(versions, name)
			}
		} else {
			if !e.IsDir() && strings.HasSuffix(name, ".md") {
				versions = append(versions, strings.TrimSuffix(name, ".md"))
			}
		}
	}
	// Ensure archived version present
	found := false
	for _, v := range versions {
		if v == version {
			found = true
			break
		}
	}
	if !found {
		versions = append(versions, version)
	}
	// Add current version label
	versions = append(versions, newLatestLabel)
	// Sort chronologically (dates sort naturally)
	sort.Strings(versions)
	// Inject revisions_history as YAML/TOML list
	archivedFM, _ = fm.InjectList(archivedFM, "revisions_history", versions)

	// Write archived file
	if err := os.WriteFile(archivedFile, []byte(fm.Stringify(archivedFM)), 0o644); err != nil {
		return err
	}

	// Propagate updated revisions_history to all existing archived versions
	// This ensures every historical version page has the same, up-to-date list
	entries, _ = os.ReadDir(revisionsDir)
	for _, e := range entries {
		name := e.Name()
		var targetPath string
		if isBundle {
			if !e.IsDir() { // skip files in bundle .revisions root
				continue
			}
			// bundle archived file is index.md inside version directory
			targetPath = filepath.Join(revisionsDir, name, "index.md")
		} else {
			if e.IsDir() || !strings.HasSuffix(name, ".md") {
				continue
			}
			targetPath = filepath.Join(revisionsDir, name)
		}

		// Skip if target file doesn't exist (defensive for bundles that may have assets only)
		if _, err := os.Stat(targetPath); err != nil {
			continue
		}

		// Read, update revisions_history, and write back
		data, err := os.ReadFile(targetPath)
		if err != nil {
			continue
		}
		fmParsed, err := fm.Parse(string(data))
		if err != nil {
			continue
		}
		// Migrate to list format
		fmParsed, _ = fm.InjectList(fmParsed, "revisions_history", versions)
		_ = os.WriteFile(targetPath, []byte(fm.Stringify(fmParsed)), 0o644)
	}

	// For bundles, copy all other files in the source bundle directory
	if isBundle {
		srcDir := filepath.Dir(sourceFile)
		if err := copyDirContents(srcDir, archivedDir, []string{"index.md"}); err != nil {
			return err
		}
	}

	// Update source file with current lastmod and date
	now := time.Now()
	currentDateTime := now.Format("2006-01-02T15:04:05-07:00")

	// Update lastmod and date to current time (unquoted, RFC3339 format for Hugo compatibility)
	parsed, _ = fm.InjectKVUnquoted(parsed, "lastmod", currentDateTime)
	parsed, _ = fm.InjectKVUnquoted(parsed, "date", currentDateTime)
	// Inject revisions_history into current page as list
	parsed, _ = fm.InjectList(parsed, "revisions_history", versions)

	if err := os.WriteFile(sourceFile, []byte(fm.Stringify(parsed)), 0o644); err != nil {
		return err
	}

	// Log operations
	ops := lastOp{Timestamp: time.Now().Format(time.RFC3339)}
	if isBundle {
		ops.Changes = append(ops.Changes, change{Source: filepath.Dir(sourceFile), Target: archivedDir, Action: "copy"})
	} else {
		ops.Changes = append(ops.Changes, change{Source: sourceFile, Target: archivedFile, Action: "copy"})
	}
	ops.Changes = append(ops.Changes, change{Source: sourceFile, Target: sourceFile, Action: "write"})
	logPath := filepath.Join(config.LogDirectory, "last_op.json")
	jb, _ := json.MarshalIndent(ops, "", "  ")
	if err := os.WriteFile(logPath, jb, 0o644); err != nil {
		return err
	}
	return nil
}

// copyDirContents copies all files and subdirectories from src to dst.
// excludeFiles lists relative file names in src to skip (e.g., "index.md").
func copyDirContents(src, dst string, excludeFiles []string) error {
	// Build exclusion set
	exclude := map[string]struct{}{}
	for _, f := range excludeFiles {
		exclude[f] = struct{}{}
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "." {
			return nil
		}
		// Skip excluded top-level files
		if _, ok := exclude[rel]; ok {
			return nil
		}
		// Compute target path
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		// Ensure parent exists
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		// Copy file bytes
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// extractDocumentDate extracts the publish date to form version (prefers date, then lastmod)
func extractDocumentDate(frontMatter fm.FrontMatter, format string) string {
	// Prefer date field (publish date)
	date := fm.GetValue(frontMatter, "date")
	if date != "" {
		if t, err := parseDate(date); err == nil {
			return t.Format(format)
		}
	}

	// Fallback to lastmod
	lastmod := fm.GetValue(frontMatter, "lastmod")
	if lastmod != "" {
		if t, err := parseDate(lastmod); err == nil {
			return t.Format(format)
		}
	}

	// If no date found or parse failed, use current time
	return time.Now().Format(format)
}

// parseDate tries to parse date string in common formats
func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func extractBaseURL(f fm.FrontMatter, bundleDir string) string {
	// First check if url field exists in front matter
	existingURL := fm.GetValue(f, "url")
	if existingURL != "" {
		// Use existing URL, ensure it ends with /
		if !strings.HasSuffix(existingURL, "/") {
			existingURL += "/"
		}
		return existingURL
	}

	// Try to get URL from Hugo permalink rules (respects Hugo config)
	if hugoURL, err := getPageURLFromHugo(bundleDir, f); err == nil && hugoURL != "" {
		if !strings.HasSuffix(hugoURL, "/") {
			hugoURL += "/"
		}
		return hugoURL
	}

	// Check for slug field
	slug := fm.GetValue(f, "slug")
	if slug != "" {
		// Derive from slug (assume section from path)
		section := extractSection(bundleDir)
		if section != "" {
			return fmt.Sprintf("/%s/%s/", section, slug)
		}
		return fmt.Sprintf("/%s/", slug)
	}

	// Fallback: derive from bundle directory path
	// e.g., content/posts/my-post -> /posts/my-post/
	parts := strings.Split(bundleDir, string(filepath.Separator))
	var afterContent []string
	found := false
	for _, p := range parts {
		if found {
			afterContent = append(afterContent, p)
		}
		if p == "content" {
			found = true
		}
	}
	return "/" + strings.Join(afterContent, "/") + "/"
}

func extractSection(bundleDir string) string {
	// Extract section from path like content/posts/demo -> posts
	parts := strings.Split(bundleDir, string(filepath.Separator))
	for i, p := range parts {
		if p == "content" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
