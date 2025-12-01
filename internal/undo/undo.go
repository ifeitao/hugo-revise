package undo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/ifeitao/hugo-revise/internal/config"
)

type change struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Action string `json:"action"`
}

type lastOp struct {
	Changes []change `json:"changes"`
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
	// minimal undo: remove copied version directory
	for _, c := range op.Changes {
		if c.Action == "copy" {
			_ = os.RemoveAll(c.Target)
		}
	}
	_ = os.Remove(logPath)
	return nil
}
