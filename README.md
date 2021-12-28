## Watch

🔎 **gowatch**

Live reload for go apps

### Motivation

I had no app to live reload my Go programs.

### Usage

Install

```sh
go install github.com/gelfand/gowatch
```

```sh
◆ go ❯❯❯ gowatch -h
Usage of gowatch:
  -cmd string
        command to run
  -path string
        path to watch
```

For example: 

```sh
◆ go ❯❯❯ gowatch -path ./playground -cmd="cal"
   December 2021
Su Mo Tu We Th Fr Sa
          1  2  3  4
 5  6  7  8  9 10 11
12 13 14 15 16 17 18
19 20 21 22 23 24 25
26 27 28 29 30 31
```

### TODO

- [ ] - Glob pattern matching.

- [ ] - Tests.




