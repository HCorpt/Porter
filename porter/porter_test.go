package porter_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMkDir(t *testing.T) {
	wd, _ := os.Getwd()
	pp := filepath.Join(wd, "hello")
	if err := os.Mkdir(pp, 0755); err != nil && os.IsExist(err) {
		fmt.Printf("create dir with exist error %s\n", err.Error())
	} else {
		fmt.Printf("create dir filed\n")
	}
}
