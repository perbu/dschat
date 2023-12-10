package main

import (
	"context"
	"fmt"
	"github.com/anthdm/hollywood/actor"
	"github.com/perbu/dschat/sshnode"
	"os"
	"os/signal"
)

func main() {
	err := realMain()
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func realMain() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	engine, err := actor.NewEngine()
	if err != nil {
		return fmt.Errorf("actor.NewEngine: %w", err)
	}
	// create a secure node.
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("os.Hostname: %w", err)
	}
	secPid := engine.Spawn(
		sshnode.NewSshNode,
		hostname,
	)

	<-ctx.Done()
	engine.Poison(secPid).Wait()

	return nil
}
