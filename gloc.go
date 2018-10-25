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

func getTtyHeightWidth() (int, int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	var height int
	var width int
	if err != nil {
		height = 1
		width = 50
	}
	hw := strings.Split(string(out), " ")
	height, err = strconv.Atoi(hw[0])
	if err != nil {
		fmt.Println(err)
		height = 1
	}
	width, err = strconv.Atoi(hw[1][:len(hw[1])-1])
	if err != nil {
		fmt.Println(err)
		width = 50
	}
	return height, width
}

func printStatus(dirStatusList map[string]dirStatus, completion chan dirStatus) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()

	fadedOutput := color.New(color.FgCyan)
	for {
		ds := <-completion
		project := getProjectFromPath(ds.path)
		fmt.Printf("\x1b[2K")
		if ds.err {
			fmt.Println(red("✖"), white(project))
		} else {
			fmt.Println(green("✔"), white(project))
		}
		dirStatusList[ds.path] = ds
		count, repos := countRemaining(dirStatusList)
		_, width := getTtyHeightWidth()
		finalOutput := strconv.Itoa(len(dirStatusList)-count) + "| " + strings.Join(repos, ", ")
		if width < 5 {
			finalOutput = ""
		} else if len(finalOutput) > width {
			finalOutput = finalOutput[:width-4] + "..."
		}
		fadedOutput.Printf(finalOutput + "\r")
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
	root := "."

	if len(os.Args) > 2 {
		root = os.Args[2]
		command = strings.Split(os.Args[1], " ")
	} else if len(os.Args) > 1 {
		command = strings.Split(os.Args[1], " ")
	}

	root = expandDir(root)
	gfiles := getGitDirs(root)

	if len(gfiles) == 0 {
		fmt.Printf("No repos found in '%s'\n", root)
		return
	}

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
