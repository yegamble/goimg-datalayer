Instead of changing `fullPath`'s signature, I can change `fullPath` to return `(string, error)` and change all callers to handle the error.
Let's change `fullPath` method to look like this:

```go
// fullPath returns the full filesystem path for a storage key and ensures it stays within the basePath.
func (s *Storage) fullPath(key string) (string, error) {
	cleanPath := filepath.Clean(filepath.Join(s.basePath, key))

	// Ensure the final path stays within bounds to prevent partial directory name bypasses
	baseDir := filepath.Clean(s.basePath)
	if !strings.HasPrefix(cleanPath, baseDir+string(filepath.Separator)) && cleanPath != baseDir {
		return "", fmt.Errorf("%w: path escapes base directory", errPathTraversal)
	}

	return cleanPath, nil
}
```

Wait, then I need to update all usages of `s.fullPath(key)` to handle the error:

```go
	fullPath, err := s.fullPath(key)
	if err != nil {
		return err
	}
```

This ensures proper path traversal prevention using `filepath.Join`, `filepath.Clean` and `strings.HasPrefix` checking bounds as highlighted in `.jules/sentinel.md` memory note:
"When validating file paths to prevent path traversal, do not rely solely on `strings.Contains(path, "..")`. Always resolve the path using `filepath.Join` and sanitize it with `filepath.Clean`. Ensure the final path stays within bounds by checking `strings.HasPrefix(cleanPath, filepath.Clean(baseDir) + string(filepath.Separator))` to prevent partial directory name bypasses."
