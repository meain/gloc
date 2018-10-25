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

## Development

### Build

```
dep ensure
go build gloc.go
```

## TODO

- [ ] Option to show output and time
