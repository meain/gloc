package main

import (
	"flag"
	"fmt"
	"log"
	"math"
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
func getGitDirs(root string, includeNonGit bool) []string {
	// check for / at end
	if root[len(root)-1] != '/' {
		root = root + "/"
	}

	if includeNonGit {
		file, err := os.Open(root)
		if err != nil {
			log.Fatal(err)
		}
		files, err := file.Readdirnames(-1)
		for f := range files {
			files[f] = root + files[f]
		}
		if err != nil {
			log.Fatal(err)
		}
		return files
	}

	// if git dirs only
	files, err := filepath.Glob(root + "*/.git")
	if err != nil {
		log.Fatal(err)
	}
	for index := range files {
		file := files[index][:len(files[index])-5]
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
			completion <- dirStatus{path, true, true, string(output)}
		} else {
			completion <- dirStatus{path, true, false, string(output)}
		}
	} else {
		cmd := exec.Command(command[0])
		cmd.Dir = path
		output, err := cmd.CombinedOutput()

		if err != nil {
			completion <- dirStatus{path, true, true, string(output)}
		} else {
			completion <- dirStatus{path, true, false, string(output)}
		}
	}
}

type dirStatus struct {
	path   string
	done   bool
	err    bool
	output string
}

func getProjectFromPath(path string) string {
	pathChunks := strings.Split(path, "/")
	return pathChunks[len(pathChunks)-1]
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
		return height, width
	}
	hw := strings.Split(string(out), " ")
	height, err = strconv.Atoi(hw[0])
	if err != nil {
		height = 1
	}
	width, err = strconv.Atoi(hw[1][:len(hw[1])-1])
	if err != nil {
		width = 50
	}
	return height, width
}

func printStatus(dirStatusList map[string]dirStatus, completion chan dirStatus, showOutput bool) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()

	redbg := color.New(color.BgRed, color.FgBlack).SprintFunc()
	greenbg := color.New(color.BgGreen, color.FgBlack).SprintFunc()

	fadedOutput := color.New(color.FgCyan)
	for {
		ds := <-completion
		project := getProjectFromPath(ds.path)
		fmt.Printf("\x1b[2K")
		if ds.err {
			if showOutput {
				fmt.Println(redbg(" ✖ " + project + " "))
				fmt.Println(ds.output)
			} else {
				fmt.Println(red("✖"), white(project))
			}
		} else {
			if showOutput {
				fmt.Println(greenbg(" ✔ " + project + " "))
				fmt.Println(ds.output)
			} else {
				fmt.Println(green("✔"), white(project))
			}
		}
		dirStatusList[ds.path] = ds
		count, repos := countRemaining(dirStatusList)

		if count == len(dirStatusList) {
			break
		}

		_, width := getTtyHeightWidth()
		width = int(math.Min(float64(width), 100))
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

func showHelp() {
	fmt.Println("Usage: gloc \"<command>\" <path>")
	fmt.Println("Optional parameters")
	flag.PrintDefaults()
	fmt.Println("\nExample: gloc \"git fetch\" ~/Documents/Projects")
}

func main() {
	var wg sync.WaitGroup

	var command []string
	root := "."

	var help bool
	var printOutput bool
	var includeNonGit bool

	flag.Usage = showHelp
	flag.BoolVar(&help, "help", false, "show help")
	flag.BoolVar(&printOutput, "output", false, "show output of the command")
	flag.BoolVar(&includeNonGit, "all-dirs", false, "show output of the command")
	flag.Parse()

	if help {
		showHelp()
		return
	}

	args := flag.Args()
	if len(args) > 1 {
		root = args[1]
		command = strings.Split(args[0], " ")
	} else if len(flag.Args()) > 0 {
		command = strings.Split(args[0], " ")
	} else {
		red := color.New(color.FgRed).SprintFunc()
		fmt.Println(red("ERROR: No command provided"))
		showHelp()
		return
	}

	root = expandDir(root)
	gfiles := getGitDirs(root, includeNonGit)

	if len(gfiles) == 0 {
		fmt.Printf("No repos found in '%s'\n", root)
		return
	}

	dirStatusList := make(map[string]dirStatus)
	for _, file := range gfiles {
		dirStatusList[file] = dirStatus{file, false, false, ""}
	}

	completion := make(chan dirStatus)

	go func(dirStatusList map[string]dirStatus, completion chan dirStatus) {
		wg.Add(1)
		printStatus(dirStatusList, completion, printOutput)
		wg.Done()
	}(dirStatusList, completion)

	for _, gdir := range gfiles {
		wg.Add(1)
		go func(gdir string) {
			defer wg.Done()
			runCommand(gdir, command, completion)
		}(gdir)
	}

	wg.Wait()
}
