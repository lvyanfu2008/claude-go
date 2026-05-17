package slashresolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/commands/featuregates"

	"go.starlark.net/starlark"
)

func starlarkEnvBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	var defaultVal string
	if err := starlark.UnpackArgs("env", args, kwargs, "key", &key, "default?", &defaultVal); err != nil {
		return starlark.None, err
	}
	v := os.Getenv(key)
	if v == "" {
		v = defaultVal
	}
	return starlark.String(v), nil
}

func starlarkCwdBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("cwd", args, kwargs); err != nil {
		return starlark.None, err
	}
	sctx := getSctx(thread)
	return starlark.String(sctx.Cwd), nil
}

func starlarkSessionIDBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("session_id", args, kwargs); err != nil {
		return starlark.None, err
	}
	sctx := getSctx(thread)
	return starlark.String(sctx.SessionID), nil
}

func starlarkSessionMemoryBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("session_memory", args, kwargs); err != nil {
		return starlark.None, err
	}
	sctx := getSctx(thread)
	return starlark.String(sctx.SessionMemory), nil
}

func starlarkUserTypeBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("user_type", args, kwargs); err != nil {
		return starlark.None, err
	}
	return starlark.String(os.Getenv("USER_TYPE")), nil
}

func starlarkIsDemoBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("is_demo", args, kwargs); err != nil {
		return starlark.None, err
	}
	v := strings.ToLower(os.Getenv("IS_DEMO"))
	return starlark.Bool(v == "1" || v == "true" || v == "yes"), nil
}

func starlarkFeatureEnabledBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs("feature_enabled", args, kwargs, "name", &name); err != nil {
		return starlark.None, err
	}
	return starlark.Bool(featuregates.Feature(name)), nil
}

func starlarkReadFileBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pathArg string
	if err := starlark.UnpackArgs("read_file", args, kwargs, "path", &pathArg); err != nil {
		return starlark.None, err
	}
	sctx := getSctx(thread)
	resolved, err := safeResolvePath(sctx.Cwd, pathArg)
	if err != nil {
		return starlark.None, fmt.Errorf("read_file: %w", err)
	}
	b, err := safeReadFile(resolved)
	if err != nil {
		return starlark.None, fmt.Errorf("read_file: %w", err)
	}
	return starlark.String(string(b)), nil
}

func starlarkFileExistsBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pathArg string
	if err := starlark.UnpackArgs("file_exists", args, kwargs, "path", &pathArg); err != nil {
		return starlark.None, err
	}
	sctx := getSctx(thread)
	resolved, err := safeResolvePath(sctx.Cwd, pathArg)
	if err != nil {
		return starlark.False, nil
	}
	_, err = os.Stat(resolved)
	return starlark.Bool(err == nil), nil
}

func starlarkListDirBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pathArg string
	if err := starlark.UnpackArgs("list_dir", args, kwargs, "path", &pathArg); err != nil {
		return starlark.None, err
	}
	sctx := getSctx(thread)
	resolved, err := safeResolvePath(sctx.Cwd, pathArg)
	if err != nil {
		return starlark.None, fmt.Errorf("list_dir: %w", err)
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return starlark.None, fmt.Errorf("list_dir: %w", err)
	}
	names := make([]starlark.Value, 0, len(entries))
	for _, e := range entries {
		names = append(names, starlark.String(e.Name()))
	}
	return starlark.NewList(names), nil
}

func starlarkArgsParsedBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// Starlark scripts receive args directly as the first parameter to resolve().
	// Use args.split() in Starlark instead.
	return starlark.NewList(nil), nil
}

func starlarkSkillRootBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("skill_root", args, kwargs); err != nil {
		return starlark.None, err
	}
	sctx := getSctx(thread)
	return starlark.String(sctx.SkillRoot), nil
}

func starlarkSessionDebugLogPathBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs("session_debug_log_path", args, kwargs); err != nil {
		return starlark.None, err
	}
	sctx := getSctx(thread)
	path := debugLogPath(sctx.SessionID)
	return starlark.String(path), nil
}

func starlarkStrContainsBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s, sub string
	if err := starlark.UnpackArgs("str_contains", args, kwargs, "s", &s, "sub", &sub); err != nil {
		return starlark.None, err
	}
	return starlark.Bool(strings.Contains(s, sub)), nil
}

func starlarkStrHasPrefixBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s, prefix string
	if err := starlark.UnpackArgs("str_has_prefix", args, kwargs, "s", &s, "prefix", &prefix); err != nil {
		return starlark.None, err
	}
	return starlark.Bool(strings.HasPrefix(s, prefix)), nil
}

func starlarkStrHasSuffixBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s, suffix string
	if err := starlark.UnpackArgs("str_has_suffix", args, kwargs, "s", &s, "suffix", &suffix); err != nil {
		return starlark.None, err
	}
	return starlark.Bool(strings.HasSuffix(s, suffix)), nil
}

func starlarkStrJoinBuiltin(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var list *starlark.List
	var sep string
	if err := starlark.UnpackArgs("str_join", args, kwargs, "list", &list, "sep", &sep); err != nil {
		return starlark.None, err
	}
	n := list.Len()
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		s, ok := starlark.AsString(list.Index(i))
		if !ok {
			return starlark.None, fmt.Errorf("str_join: element %d is not a string", i)
		}
		parts[i] = s
	}
	return starlark.String(strings.Join(parts, sep)), nil
}

// safeResolvePath resolves a relative path against base and rejects path traversal.
func safeResolvePath(base, rel string) (string, error) {
	if base == "" {
		base = "."
	}
	resolved := filepath.Clean(filepath.Join(base, rel))
	baseClean := filepath.Clean(base)
	if !strings.HasPrefix(resolved, baseClean+string(filepath.Separator)) && resolved != baseClean {
		return "", fmt.Errorf("path traversal not allowed: %s", rel)
	}
	return resolved, nil
}

func safeReadFile(path string) ([]byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.Size() > starlarkMaxFileRead {
		return nil, fmt.Errorf("file exceeds max read size (%d bytes)", starlarkMaxFileRead)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > starlarkMaxFileRead {
		return b[:starlarkMaxFileRead], nil
	}
	return b, nil
}
