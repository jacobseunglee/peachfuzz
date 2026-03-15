package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sliver-client/internal/client"
	"sliver-client/internal/state"
	"sliver-client/internal/tracker"
	"sliver-client/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	configPath := flag.String("config", "", "path to sliver client config file (required)")
	workers := flag.Int("workers", 5, "max concurrent dispatch workers")
	refresh := flag.Duration("refresh", 5*time.Second, "session/beacon refresh interval")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: -config flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Connect to Sliver server
	rpc, conn, err := client.Connect(*configPath)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Initialize shared state
	appState := state.New(rpc, *refresh)
	go appState.StartRefresh()

	// Initialize beacon task tracker
	taskTracker := tracker.New(rpc)
	go taskTracker.Start()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		appState.Stop()
		taskTracker.Stop()
		conn.Close()
		os.Exit(0)
	}()

	// Launch TUI
	model := tui.NewApp(rpc, appState, taskTracker, *workers)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}

	appState.Stop()
	taskTracker.Stop()
}
