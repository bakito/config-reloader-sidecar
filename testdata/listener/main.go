package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, os.Interrupt, syscall.SIGTERM)
	running := true
	go func() {
		for sig := range c {
			switch sig {
			case os.Interrupt:
				running = false
			case syscall.SIGTERM:
				running = false
			case syscall.SIGHUP:
				println()
				log.Printf("Got A HUP Signal!")
			}
		}
	}()
	for {
		time.Sleep(1000 * time.Millisecond)
		fmt.Print(">")
		if !running {
			println()
			log.Printf("Terminating ....")
			return
		}
	}
}
