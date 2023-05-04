package utils

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// Return relative dir
func RecurseListFiles(dir string) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	files := []string{}
	cb := func(path string, f fs.FileInfo, err error) error {
		if !f.IsDir() {
			files = append(files, path[len(absDir)+1:])
		}
		return nil
	}
	err = filepath.Walk(absDir, cb)
	return files, err
}

func CopyFiles(dst string, src string) (int64, error) {
	err := os.MkdirAll(path.Dir(dst), 0755)
	if err != nil {
		return 0, err
	}
	dstWriter, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0755) // ignore_security_alert FILE_OPER
	defer dstWriter.Close()
	if err != nil {
		return 0, err
	}
	srcReader, err := os.OpenFile(src, os.O_RDONLY, 0755) // ignore_security_alert_wait_for_fix FILE_OPER
	defer srcReader.Close()
	if err != nil {
		return 0, err
	}
	return io.Copy(dstWriter, srcReader)
}

func WriteJsonFile(path string, obj interface{}, perm os.FileMode) error {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, perm)
}
