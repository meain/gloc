package main

import (
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
)

// Gets directories which have .git in them
func getGitDirs(root string) []string {
	// check for / at end
	if root[len(root)-1] != '/' {
		root = root + "/"
	}

	files, err := filepath.Glob(root + "*/.git")
	if err != nil {
		log.Fatal(err)
	}
	for index := range files {
		file := files[index][:len(files[index])-4]
		files[index] = file
	}

	return files
}

func runCommand(path string) {
	errorOutput := color.New(color.FgRed)
	okOutput := color.New(color.FgGreen)
	cmd := exec.Command("git", "fetch")
	cmd.Dir = path
	_, err := cmd.CombinedOutput()

	pathChunks := strings.Split(path, "/")
	project := pathChunks[len(pathChunks)-2]
	if err != nil {
		errorOutput.Println(project)
	} else {
		okOutput.Println(project)
	}
}

func main() {
	var wg sync.WaitGroup
	root := "/Users/meain/Documents/Projects/others/clones"
	gfiles := getGitDirs(root)

	for _, gdir := range gfiles {
		wg.Add(1)
		go func(gdir string) {
            defer wg.Done()
			runCommand(gdir)
		}(gdir)
	}

	wg.Wait()
}
