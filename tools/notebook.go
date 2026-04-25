package tools

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"goc/tools/localtools"
)

func randomCellID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NotebookEditFromJSON edits a .ipynb (nbformat) notebook; subset parity with TS NotebookEditTool.
func NotebookEditFromJSON(raw []byte, roots []string) (string, bool, error) {
	var in struct {
		NotebookPath string `json:"notebook_path"`
		CellID       string `json:"cell_id"`
		NewSource    string `json:"new_source"`
		CellType     string `json:"cell_type"`
		EditMode     string `json:"edit_mode"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	mode := strings.TrimSpace(in.EditMode)
	if mode == "" {
		mode = "replace"
	}
	if mode != "replace" && mode != "insert" && mode != "delete" {
		return "", true, fmt.Errorf("edit_mode must be replace, insert, or delete")
	}
	if mode == "insert" && strings.TrimSpace(in.CellType) == "" {
		return "", true, fmt.Errorf("cell_type is required when edit_mode=insert")
	}
	if mode != "insert" && strings.TrimSpace(in.CellID) == "" {
		return "", true, fmt.Errorf("cell_id is required unless edit_mode=insert")
	}

	abs, err := localtools.ResolveUnderRoots(in.NotebookPath, roots)
	if err != nil {
		return "", true, err
	}
	if strings.ToLower(filepath.Ext(abs)) != ".ipynb" {
		return "", true, fmt.Errorf("file must be a Jupyter notebook (.ipynb)")
	}

	data, err := readFileLimited(abs, 32<<20)
	if err != nil {
		return "", true, err
	}
	var nb map[string]any
	if err := json.Unmarshal(data, &nb); err != nil {
		return "", true, fmt.Errorf("notebook is not valid JSON: %w", err)
	}
	rawCells, ok := nb["cells"]
	if !ok {
		return "", true, fmt.Errorf("notebook missing cells array")
	}
	cells, ok := rawCells.([]any)
	if !ok {
		return "", true, fmt.Errorf("notebook cells must be an array")
	}

	origBytes, _ := json.MarshalIndent(nb, "", "  ")
	origStr := string(origBytes)

	var deletedCellType string
	switch mode {
	case "replace", "delete":
		idx, err := findCellIndexByID(cells, in.CellID)
		if err != nil {
			return "", true, err
		}
		if mode == "delete" {
			if cm, ok := cells[idx].(map[string]any); ok {
				if s, ok := cm["cell_type"].(string); ok {
					deletedCellType = s
				}
			}
			nb["cells"] = append(cells[:idx], cells[idx+1:]...)
		} else {
			cell, ok := cells[idx].(map[string]any)
			if !ok {
				return "", true, fmt.Errorf("cell %d is not an object", idx)
			}
			setCellSource(cell, in.NewSource)
			if ct := strings.TrimSpace(in.CellType); ct != "" {
				cell["cell_type"] = ct
			}
		}
	case "insert":
		ct := strings.TrimSpace(in.CellType)
		if ct != "code" && ct != "markdown" {
			return "", true, fmt.Errorf("cell_type must be code or markdown")
		}
		nm := newNotebookCell(ct, in.NewSource)
		newCell := any(nm)
		insertAfter := strings.TrimSpace(in.CellID)
		if insertAfter == "" {
			nb["cells"] = append([]any{newCell}, cells...)
		} else {
			idx, err := findCellIndexByID(cells, insertAfter)
			if err != nil {
				return "", true, err
			}
			out := make([]any, 0, len(cells)+1)
			out = append(out, cells[:idx+1]...)
			out = append(out, newCell)
			out = append(out, cells[idx+1:]...)
			nb["cells"] = out
		}
		in.CellID = CellIDFromCell(nm)
	}

	ctOut := strings.TrimSpace(in.CellType)
	if ctOut == "" && mode == "delete" && deletedCellType != "" {
		ctOut = deletedCellType
	}
	if ctOut == "" {
		rawCells, _ := nb["cells"].([]any)
		id := strings.TrimSpace(in.CellID)
		if id != "" {
			if idx, err := findCellIndexByID(rawCells, id); err == nil {
				if cm, ok := rawCells[idx].(map[string]any); ok {
					if s, ok := cm["cell_type"].(string); ok && s != "" {
						ctOut = s
					}
				}
			}
		}
		if ctOut == "" {
			ctOut = "code"
		}
	}

	outBytes, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		return "", true, err
	}
	if err := writeFileAtomic(abs, outBytes, 0o644); err != nil {
		return "", true, err
	}

	lang := "unknown"
	if meta, ok := nb["metadata"].(map[string]any); ok {
		if k, ok := meta["kernelspec"].(map[string]any); ok {
			if l, ok := k["language"].(string); ok {
				lang = l
			}
		}
	}
	cid := strings.TrimSpace(in.CellID)
	resp := map[string]any{
		"new_source":    in.NewSource,
		"cell_id":       cid,
		"cell_type":     ctOut,
		"language":      lang,
		"edit_mode":     mode,
		"error":         "",
		"notebook_path": abs,
		"original_file": origStr,
		"updated_file":  string(outBytes),
	}
	b, _ := json.Marshal(map[string]any{"data": resp})
	return string(b), false, nil
}

// CellIDFromCell extracts the cell ID from a notebook cell's metadata.
func CellIDFromCell(cell map[string]any) string {
	if meta, ok := cell["metadata"].(map[string]any); ok {
		if id, ok := meta["id"].(string); ok && id != "" {
			return id
		}
	}
	return ""
}

// CellSourceText reconstructs the source text of a notebook cell from its source array.
func CellSourceText(cell map[string]any) string {
	raw, ok := cell["source"]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, line := range v {
			if s, ok := line.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "")
	case []string:
		return strings.Join(v, "")
	}
	return ""
}

// NotebookEditFromEdit converts a FileEdit request targeting .ipynb to a NotebookEdit call.
// This is the redirect path: when Edit detects .ipynb, it routes here instead of erroring.
func NotebookEditFromEdit(absPath, oldString, newString string, replaceAll bool, roots []string) (string, bool, error) {
	_ = replaceAll // .ipynb edits always replace first occurrence within a single cell
	data, err := readFileLimited(absPath, 32<<20)
	if err != nil {
		return "", true, err
	}
	var nb map[string]any
	if err := json.Unmarshal(data, &nb); err != nil {
		return "", true, fmt.Errorf("notebook is not valid JSON: %w", err)
	}
	rawCells, ok := nb["cells"]
	if !ok {
		return "", true, fmt.Errorf("notebook missing cells array")
	}
	cells, ok := rawCells.([]any)
	if !ok {
		return "", true, fmt.Errorf("notebook cells must be an array")
	}

	for _, c := range cells {
		cell, ok := c.(map[string]any)
		if !ok {
			continue
		}
		cellSource := CellSourceText(cell)
		if strings.Contains(cellSource, oldString) {
			cellID := CellIDFromCell(cell)
			newSource := strings.Replace(cellSource, oldString, newString, 1)

			nbInput := map[string]any{
				"notebook_path": absPath,
				"cell_id":       cellID,
				"new_source":    newSource,
				"cell_type":     "",
				"edit_mode":     "replace",
			}
			nbRaw, _ := json.Marshal(nbInput)
			return NotebookEditFromJSON(nbRaw, roots)
		}
	}

	return "", true, fmt.Errorf("String to replace not found in notebook cells.\nString: %s", oldString)
}

func findCellIndexByID(cells []any, id string) (int, error) {
	id = strings.TrimSpace(id)
	for i, c := range cells {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if CellIDFromCell(cm) == id {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no cell with id %q", id)
}

func setCellSource(cell map[string]any, src string) {
	lines := strings.Split(src, "\n")
	quoted := make([]string, len(lines))
	for i, l := range lines {
		quoted[i] = l + "\n"
	}
	cell["source"] = quoted
}

func newNotebookCell(cellType, src string) map[string]any {
	cell := map[string]any{
		"cell_type": cellType,
		"metadata": map[string]any{
			"id": randomCellID(),
		},
	}
	setCellSource(cell, src)
	if cellType == "code" {
		cell["outputs"] = []any{}
		cell["execution_count"] = nil
	}
	return cell
}
