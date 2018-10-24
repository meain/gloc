package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
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

func runCommand(path string, command []string, completion chan dirStatus) {
	// TODO: define cmd outside, not sure how to set type
	if len(command) > 1 {
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Dir = path
		output, err := cmd.CombinedOutput()

		if err != nil {
			completion <- dirStatus{path, true, true, output}
		} else {
			completion <- dirStatus{path, true, false, output}
		}
	} else {
		cmd := exec.Command(command[0])
		cmd.Dir = path
		output, err := cmd.CombinedOutput()

		if err != nil {
			completion <- dirStatus{path, true, true, output}
		} else {
			completion <- dirStatus{path, true, false, output}
		}
	}
}

type dirStatus struct {
	path   string
	done   bool
	err    bool
	output []byte
}

func getProjectFromPath(path string) string {
	pathChunks := strings.Split(path, "/")
	return pathChunks[len(pathChunks)-2]
}

func countRemaining(dirStatusList map[string]dirStatus) (int, []string) {
	count := 0
	var repos []string
	for _, ds := range dirStatusList {
		if ds.done {
			count++
		} else {
			repos = append(repos, getProjectFromPath(ds.path))
		}
	}
	return count, repos
}

func printStatus(dirStatusList map[string]dirStatus, completion chan dirStatus) {
	errorOutput := color.New(color.FgRed)
	okOutput := color.New(color.FgGreen)
	fadedOutput := color.New(color.FgCyan)
	for {
		ds := <-completion
		project := getProjectFromPath(ds.path)
		fmt.Printf("\x1b[2K")
		if ds.err {
			errorOutput.Println(project)
		} else {
			okOutput.Println(project)
		}
		dirStatusList[ds.path] = ds
		count, repos := countRemaining(dirStatusList)
		if len(repos) > 4 {
			repos = repos[:4]
		}
		fadedOutput.Printf(strconv.Itoa(len(dirStatusList)-count) + "| " + strings.Join(repos, ", ") + "\r")
	}
}

func expandDir(path string) string {
	usr, _ := user.Current()
	dir := usr.HomeDir
	if path == "~" {
		path = dir
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(dir, path[2:])
	}
	return path
}

func main() {
	var wg sync.WaitGroup

	command := []string{"git", "fetch"}
	root := "./"

	if len(os.Args) > 2 {
		root = os.Args[2]
		command = strings.Split(os.Args[1], " ")
	} else if len(os.Args) > 1 {
		command = strings.Split(os.Args[1], " ")
	}

	root = expandDir(root)
	gfiles := getGitDirs(root)

	dirStatusList := make(map[string]dirStatus)
	for _, file := range gfiles {
		dirStatusList[file] = dirStatus{file, false, false, []byte{}}
	}

	completion := make(chan dirStatus)

	go printStatus(dirStatusList, completion)

	for _, gdir := range gfiles {
		wg.Add(1)
		go func(gdir string) {
			defer wg.Done()
			runCommand(gdir, command, completion)
		}(gdir)
	}

	wg.Wait()
}
