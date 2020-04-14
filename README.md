# gloc

Run a shell command in all the git repos in a directory.

![gif](https://i.imgur.com/Ss2B2kR.gif)


> Idea stolen from [`fabiospampinato/autogit`](https://github.com/fabiospampinato/autogit)


## Install

### macOS

```
brew tap meain/homebrew-meain
brew install meain/homebrew-meain/gloc
```

### Manual

Download the binary from the [release page](https://github.com/meain/gloc/releases).

## Usage

```
gloc "git fetch" "./path/to/folder"
```

You can provide two other flags

- `--output` to show the output of the command
- `--all-dirs` to do the command on all dirs and not just git projects
