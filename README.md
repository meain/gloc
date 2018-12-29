# gloc

Run a shell command in all the git repos in a directory.

![gif](https://i.imgur.com/Ss2B2kR.gif)


> Idea stolen from [`fabiospampinato/autogit`](https://github.com/fabiospampinato/autogit)


## Install

```
dep ensure
go install
```

## Usage

```
./gloc "git fetch" "./path/to/folder"
```

You can provide two other flags

- `--output` to show the output of the command
- `--all-dirs` to do the command on all dirs and not just git projects

## Development

### Build

```
dep ensure
go build gloc.go
```
