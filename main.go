package main

import (
	"fmt"

	"github.com/HCorpt/porter/utils"
)

func main() {
	files, err := utils.RecurseListFiles("./")
	if err != nil {
		fmt.Printf("recurse list files with errors: %s", err.Error())
	}
	for _, file := range files {
		fmt.Println(file)
	}

	n, err := utils.CopyFiles("./parrent/back.go", "main.go")
	if err != nil {
		fmt.Printf("copy file error %s\n", err.Error())
	}
	fmt.Printf("copy file %d", n)
}
