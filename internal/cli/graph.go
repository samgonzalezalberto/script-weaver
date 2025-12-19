package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
)

type graphFile struct {
	Tasks []core.Task `json:"tasks"`
	Edges []dag.Edge  `json:"edges"`
}

// LoadGraphFromFile reads and parses the graph definition at path.
//
// Current supported format: JSON.
//
// The loader is deterministic:
//   - Disallows unknown fields (to avoid silent divergence).
//   - Does not consult environment variables.
func LoadGraphFromFile(path string) (*dag.TaskGraph, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read graph: %w", err)
	}
	var gf graphFile
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&gf); err != nil {
		return nil, fmt.Errorf("parse graph json: %w", err)
	}
	// Ensure there is no trailing garbage (including a second JSON value).
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("parse graph json: trailing data")
		}
		return nil, fmt.Errorf("parse graph json: %w", err)
	}
	if len(gf.Tasks) == 0 {
		return nil, fmt.Errorf("parse graph json: no tasks")
	}
	g, err := dag.NewTaskGraph(gf.Tasks, gf.Edges)
	if err != nil {
		return nil, err
	}
	return g, nil
}
