# Foreman
It is a [foreman](https://github.com/ddollar/foreman) implementation in GO.

## Description
Foreman is a manager for [Procfile-based](https://en.wikipedia.org/wiki/Procfs) applications. Its aim is to abstract away the details of the Procfile format, and allow you to run your services directly.

## Features
- Run procfile-backed apps.
- Able to run with dependency resolution.

## Procfile
Procfile is simply `key: value` format like:
```yaml
app:
  cmd: ping -c 5 google.com 
  checks:
    cmd: sleep 1
  deps: 
      - redis
redis:
  cmd: redis-server --port 6010
  run_once: true
  checks:
    cmd: redis-cli -p 6010 ping
```
**Here** we defined two services `app` and `redis` with check commands and dependency matrix

## API
- To Get the package:
```sh
go get github.com/IslamWalid/foreman
```

- Functionality:

| Function | Description |
|----------|-------------|
| `func New(procfilePath string) (*Foreman, error)` | Parses the given procfile in the procfilePath, builds and return a new foreman object. |
| `func (f *Foreman) Start() error` | Starts the services parsed from the procfile. |
| `func (f *Foreman) Exit(exitStatus int)` | Kill all the running services and their checkers, then exits the program with the given exit status. |

## How to use
- Install foreman
```sh
go install github.com/IslamWalid/foreman/cmd/forman@latest
```
- Create the procfile with processes or services you want to run.

- Pass the procfile path as an argument to foreman: 
```sh
foreman /path/to/procfile
```
