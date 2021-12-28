package main

import (
	"context"
	"errors"
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

// errEmptyCmd returned when caller passes empty command.
var errEmptyCmd = errors.New("command is empty")

// NOTE: flock is not used, because it's all sequantial.
//
// flock is a global file lock.
// var flock = sync.Mutex{}

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

// cmdArgs is struct with command name and it's arguments.
type cmdArgs struct {
	name string
	args []string
}

// config is the configuration for the gowatch.
type config struct {
	// Path is the path to watch.
	Path string
	args cmdArgs
}

// initConfig initializes the configurations.
func initConfig() (config, error) {
	entryPath, err := filepath.Abs(*pathFlag)
	if err != nil {
		return config{}, fmt.Errorf("invlaid path: %w", err)
	}

	cmdRaw := strings.Split(*cmdFlag, " ")
	if len(cmdRaw) == 0 {
		return config{}, errEmptyCmd
	}

	cmdName := cmdRaw[0]
	var flags []string
	if len(cmdRaw) > 1 {
		flags = cmdRaw[1:]
	}

	return config{
		Path: entryPath,
		args: cmdArgs{
			name: cmdName,
			args: flags,
		},
	}, nil
}

func watcher(ctx context.Context, dirpath string, notify chan<- struct{}) {
	cycle := time.NewTicker(time.Second)
	defer cycle.Stop()

	fileInfos := make(map[string]os.FileInfo)
	err := filepath.WalkDir(dirpath, func(path string, d os.DirEntry, err error) error {
		fileInfo, err := d.Info()
		if err != nil {
			return fmt.Errorf("could not get %v info: %w", path, err)
		}
		fileInfos[path] = fileInfo
		return nil
	})
	if err != nil {
		log.Fatalf("could not start gowatch, after unsuccessful program initialization: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-cycle.C:
			changed := false
			// TODO:: replace by own implementation to not call call recursively walk dir,
			// because right not it's pretty junky implementation.
			// See: path/filepath/path.go walkDir
			filepath.WalkDir(dirpath, func(path string, d os.DirEntry, err error) error {
				fileInfo, err := d.Info()
				if err != nil {
					return err
				}
				v, ok := fileInfos[path]
				if !ok && !changed {
					changed = true
				}
				if !changed && ((v.ModTime() != fileInfo.ModTime()) || (v.Size() != fileInfo.Size())) {
					changed = true
				}

				fileInfos[path] = fileInfo
				return nil
			})

			if changed {
				go func() {
					notify <- struct{}{}
				}()
			}
		}
	}
}

func runner(ctx context.Context, args cmdArgs, notify <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-notify:
			cmdCtx, cancel := context.WithCancel(context.Background())
			cmd := exec.CommandContext(cmdCtx, args.name, args.args...)
			go func() {
				cmd.Run()
			}()

			select {
			case <-ctx.Done():
				cancel()
				return
			case <-notify:
				log.Printf("\nReloading...\n")
				cancel()
			}
		}
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := initConfig()
	if err != nil {
		log.Fatalf("could not start gowatch: %v", err)
	}

	notify := make(chan struct{}, 1)

	go watcher(ctx, cfg.Path, notify)
	runner(ctx, cfg.args, notify)
}
