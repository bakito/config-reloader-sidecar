package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	ps "github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/unix"
)

const (
	envConfigDir    = "CONFIG_DIR"
	envProcessName  = "PROCESS_NAME"
	envVerbose      = "VERBOSE"
	envReloadSignal = "RELOAD_SIGNAL"
)

func main() {
	configDir := os.Getenv(envConfigDir)
	if configDir == "" {
		log.Fatalf("mandatory env var %q is empty, exiting", envConfigDir)
	}

	processName := os.Getenv(envProcessName)
	if processName == "" {
		log.Fatalf("mandatory env var %q is empty, exiting", envProcessName)
	}

	verbose := false
	verboseFlag := os.Getenv(envVerbose)
	if verboseFlag == "true" {
		verbose = true
	}

	var reloadSignal syscall.Signal
	reloadSignalStr := os.Getenv(envReloadSignal)
	if reloadSignalStr == "" {
		log.Printf("%q is empty, defaulting to SIGHUP", envReloadSignal)
		reloadSignal = syscall.SIGHUP
	} else {
		reloadSignal = unix.SignalNum(reloadSignalStr)
		if reloadSignal == 0 {
			log.Fatalf("cannot find signal for %q: %s", envReloadSignal, reloadSignalStr)
		}
	}

	log.Printf("starting with %s=%s, %s=%s, %s=%s\n",
		envConfigDir, configDir,
		envProcessName, processName,
		envReloadSignal, reloadSignal,
	)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = watcher.Close() }()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if verbose {
					log.Println("event:", event)
				}
				if event.Op&fsnotify.Chmod != fsnotify.Chmod {
					log.Println("modified file:", event.Name)
					err := reloadProcess(processName, reloadSignal)
					if err != nil {
						log.Println("error:", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	configDirs := strings.Split(configDir, ",")
	for _, dir := range configDirs {
		err = watcher.Add(dir)
		if err != nil {
			log.Fatal(err)
		}
	}

	<-done
}

func findPID(process string) (int, error) {
	processes, err := ps.Processes()
	if err != nil {
		return -1, fmt.Errorf("failed to list processes: %w\n", err)
	}

	for _, p := range processes {
		name, err := p.Name()
		if err == nil && name == process {
			log.Printf("found executable %s (pid: %d)\n", name, p.Pid)
			return int(p.Pid), nil
		}
	}

	return -1, fmt.Errorf("no process matching %s found\n", process)
}

func reloadProcess(process string, signal syscall.Signal) error {
	pid, err := findPID(process)
	if err != nil {
		return err
	}

	err = syscall.Kill(pid, signal)
	if err != nil {
		return fmt.Errorf("could not send signal: %w\n", err)
	}

	log.Printf("signal %s sent to %s (pid: %d)\n", signal, process, pid)
	return nil
}
