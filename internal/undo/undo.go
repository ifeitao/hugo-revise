package undo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ifeitao/hugo-revise/internal/config"
	"github.com/ifeitao/hugo-revise/internal/fm"
)

type change struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Action string `json:"action"`
}

type lastOp struct {
	OriginalContent string   `json:"original_content"`
	Changes         []change `json:"changes"`
}

func Run(cfg config.Config) error {
	logPath := filepath.Join(config.LogDirectory, "last_op.json")
	b, err := os.ReadFile(logPath)
	if err != nil {
		return errors.New("no last operation to undo")
	}
	var op lastOp
	if err := json.Unmarshal(b, &op); err != nil {
		return err
	}

	// Find the source file and archived target from changes
	var sourceFile string
	var archivedTarget string
	for _, c := range op.Changes {
		if c.Action == "write" {
			sourceFile = c.Source
		}
		if c.Action == "copy" {
			archivedTarget = c.Target
		}
	}

	if sourceFile == "" || archivedTarget == "" {
		return errors.New("invalid operation log: missing source or target")
	}

	// Restore original source file content
	if op.OriginalContent != "" {
		if err := os.WriteFile(sourceFile, []byte(op.OriginalContent), 0o644); err != nil {
			return fmt.Errorf("failed to restore source file: %w", err)
		}
	} else {
		// Fallback: try to update revisions_history by removing the last version
		// This handles undo for operations logged before the OriginalContent field was added
		data, err := os.ReadFile(sourceFile)
		if err == nil {
			parsed, err := fm.Parse(string(data))
			if err == nil {
				history := fm.GetList(parsed, "revisions_history")
				if len(history) > 1 {
					// Remove the last version (current revision being undone)
					history = history[:len(history)-1]
					parsed, _ = fm.InjectList(parsed, "revisions_history", history)
					_ = os.WriteFile(sourceFile, []byte(fm.Stringify(parsed)), 0o644)
				}
			}
		}
	}

	// Remove the archived version directory/file
	if err := os.RemoveAll(archivedTarget); err != nil {
		return fmt.Errorf("failed to remove archived version: %w", err)
	}

	// Update all remaining archived versions' revisions_history
	// Determine revisions directory
	revisionsDir := ""
	if strings.HasSuffix(sourceFile, "index.md") {
		// Bundle: revisions dir is at parent/../bundlename.revisions/
		bundleDir := filepath.Dir(sourceFile)
		bundleName := filepath.Base(bundleDir)
		revisionsDir = filepath.Join(filepath.Dir(bundleDir), bundleName+".revisions")
	} else {
		// Single file: revisions dir is at same level
		baseName := strings.TrimSuffix(filepath.Base(sourceFile), ".md")
		revisionsDir = filepath.Join(filepath.Dir(sourceFile), baseName+".revisions")
	}

	if _, err := os.Stat(revisionsDir); err == nil {
		// Get updated revisions_history from restored source file
		data, err := os.ReadFile(sourceFile)
		if err == nil {
			parsed, err := fm.Parse(string(data))
			if err == nil {
				history := fm.GetList(parsed, "revisions_history")
				// Update all archived versions with the corrected history
				entries, _ := os.ReadDir(revisionsDir)
				for _, e := range entries {
					name := e.Name()
					var targetPath string
					if e.IsDir() {
						// Bundle archived version
						targetPath = filepath.Join(revisionsDir, name, "index.md")
					} else if strings.HasSuffix(name, ".md") {
						// Single file archived version
						targetPath = filepath.Join(revisionsDir, name)
					} else {
						continue
					}

					if _, err := os.Stat(targetPath); err != nil {
						continue
					}

					data, err := os.ReadFile(targetPath)
					if err != nil {
						continue
					}
					fmParsed, err := fm.Parse(string(data))
					if err != nil {
						continue
					}
					fmParsed, _ = fm.InjectList(fmParsed, "revisions_history", history)
					_ = os.WriteFile(targetPath, []byte(fm.Stringify(fmParsed)), 0o644)
				}
			}
		}
	}

	// Remove log file
	if err := os.Remove(logPath); err != nil {
		return fmt.Errorf("failed to remove log file: %w", err)
	}

	return nil
}
