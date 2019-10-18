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

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Gets directories which have .git in them
func getGitDirs(root string, includeNonGit bool, recurseInto bool) []string {
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

	if recurseInto {
		var files []string
		err := filepath.Walk(root,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.Name() == "node_modules" || info.Name() == ".git" {
					return filepath.SkipDir
				}

				if !info.IsDir() {
					return nil
				}

				file, err := os.Open(path)
				defer file.Close()
				if err != nil {
					log.Fatal(err)
				}
				df, err := file.Readdirnames(-1)
				if err != nil {
					log.Fatal(err)
				}
				if stringInSlice(".git", df) {
					files = append(files, path)
				}

				return nil
			})
		if err != nil {
			log.Fatal(err)
		}
		return files
	}

	// if git dirs only & without recursive
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

func runCommand(path string, fullCommand string, completion chan dirStatus) {
	// TODO: define cmd outside, not sure how to set type
	cmd := exec.Command("sh", "-c", fullCommand)
	cmd.Dir = path
	output, err := cmd.CombinedOutput()

	if err != nil {
		completion <- dirStatus{path, true, true, string(output)}
	} else {
		completion <- dirStatus{path, true, false, string(output)}
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

func printStatus(dirStatusList map[string]dirStatus, completion chan dirStatus, showOutput bool, ignoreEmpty bool, ignoreErrors bool) {
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
				if !(ignoreEmpty && len(ds.output) < 1) && !ignoreErrors {
					fmt.Println(redbg(" ✖ " + project + " "))
					fmt.Println(ds.output)
				}
			} else {
				fmt.Println(red("✖"), white(project))
			}
		} else {
			if showOutput {
				if !(ignoreEmpty && len(ds.output) < 1) {
					fmt.Println(greenbg(" ✔ " + project + " "))
					fmt.Println(ds.output)
				}
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

	var command string
	root := "."

	var help bool
	var printOutput bool
	var ignoreEmpty bool
	var ignoreErrors bool
	var recurseInto bool
	var includeNonGit bool
	var maxGoroutines int

	flag.Usage = showHelp
	flag.BoolVar(&help, "help", false, "show help")
	flag.BoolVar(&printOutput, "output", false, "show output of the command")
	flag.BoolVar(&ignoreEmpty, "ignore-empty", false, "ignore showing if empty output")
	flag.BoolVar(&ignoreErrors, "ignore-errors", false, "recursively fild all git dirs")
	flag.BoolVar(&recurseInto, "recurse-into", false, "recursively fild all git dirs")
	flag.BoolVar(&includeNonGit, "all-dirs", false, "show output of the command")
	flag.IntVar(&maxGoroutines, "workers", 10, "number of parallel jobs")

	flag.Parse()

	if help {
		showHelp()
		return
	}

	args := flag.Args()
	if len(args) > 1 {
		root = args[1]
		command = args[0]
	} else if len(flag.Args()) > 0 {
		command = args[0]
	} else {
		red := color.New(color.FgRed).SprintFunc()
		fmt.Println(red("ERROR: No command provided"))
		showHelp()
		return
	}

	root = expandDir(root)
	gfiles := getGitDirs(root, includeNonGit, recurseInto)

	if len(gfiles) == 0 {
		fmt.Printf("No repos found in '%s'\n", root)
		return
	}

	dirStatusList := make(map[string]dirStatus)
	for _, file := range gfiles {
		dirStatusList[file] = dirStatus{file, false, false, ""}
	}

	completion := make(chan dirStatus)
	guard := make(chan struct{}, maxGoroutines)

	go func(dirStatusList map[string]dirStatus, completion chan dirStatus) {
		wg.Add(1)
		printStatus(dirStatusList, completion, printOutput, ignoreEmpty, ignoreErrors)
		wg.Done()
	}(dirStatusList, completion)

	for _, gdir := range gfiles {
		wg.Add(1)
		guard <- struct{}{}
		go func(gdir string) {
			defer wg.Done()
			runCommand(gdir, command, completion)
			<-guard
		}(gdir)
	}

	wg.Wait()
}
