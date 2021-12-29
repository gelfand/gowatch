package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

// Flags.
var (
	cmdFlag  = flag.String("cmd", "", "command to run")
	pathFlag = flag.String("path", "", "path to watch")
)

// usage prints the cmd usage message.
func usage() {
	fmt.Fprintln(os.Stderr, "Usage of gowatch:")
	flag.PrintDefaults()
}

// watcher watches the given path for changes.
func watcher(ctx context.Context, dirpath string, notify chan<- bool, done chan<- bool) {
	cycle := time.NewTicker(time.Second)
	defer cycle.Stop()

	fileInfos := make(map[string]os.FileInfo)
	var walk func(path string)
	walk = func(path string) {
		dirs, err := os.ReadDir(path)
		if err != nil {
			// TODO: maybe ignore this
			log.Fatalf("could start gowatch, could not read directory: %v", err)
		}

		for _, d1 := range dirs {
			if strings.HasPrefix(d1.Name(), ".") {
				continue
			}

			path1 := filepath.Join(path, d1.Name())
			fileInfo, err := d1.Info()
			if err != nil {
				log.Fatal(err)
			}
			v, ok := fileInfos[path1]
			if !ok {
				go func() { notify <- true }()
				fileInfos[path1] = fileInfo
			} else {
				if v.Size() != fileInfo.Size() || v.ModTime() != fileInfo.ModTime() {
					go func() { notify <- true }()
					fileInfos[path1] = fileInfo
				}
			}

			if d1.IsDir() {
				walk(path1)
			}
		}
	}
	walk(dirpath)
	done <- true
	close(done)

	for {
		select {
		case <-ctx.Done():
			return
		case <-cycle.C:
			walk(dirpath)
		}
	}
}

// runner runs the given command.
func runner(ctx context.Context, cmdName string, args []string, notify chan bool, done chan bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-notify:
			continue
		case <-done:
		}
		break
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-notify:
			log.Printf("\nReloading...\n")
			c := exec.Command(cmdName, args...)
			r, err := c.StdoutPipe()
			if err != nil {
				log.Fatalf("could not get stdout pipe: %v", err)
			}
			c.Start()

			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-notify:
						go func() { notify <- true }()
						c.Process.Kill()
						return
					case <-time.Tick(1 * time.Second):
						buf := make([]byte, 1024)
						n, err := r.Read(buf[:])
						if err != nil {
							continue
						}
						fmt.Println(string(buf[:n]))
					}
				}
			}()
		}
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	path, err := filepath.Abs(*pathFlag)
	if err != nil {
		log.Fatalf("could not retrieve given path: %v", err)
	}
	cmdRaw := strings.Split(*cmdFlag, " ")
	if len(cmdRaw) == 0 {
		log.Fatalf("could not start gowatch, command is empty")
	}
	cmdName := cmdRaw[0]
	var args []string
	if len(cmdRaw) > 1 {
		args = cmdRaw[1:]
	}

	notify := make(chan bool, 1)
	done := make(chan bool, 1)

	go watcher(ctx, path, notify, done)
	runner(ctx, cmdName, args, notify, done)
}
