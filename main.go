package main

import (
	"fmt"

	"github.com/HCorpt/porter/utils"
)

func main() {
	files, err := utils.RecurseListFiles("/Users/Heng/Heng")
	if err != nil {
		fmt.Printf("recurse list files with errors: %s", err.Error())
	}
	for _, file := range files {
		fmt.Println(file)
	}
}
