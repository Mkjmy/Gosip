package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gosip/internal/registry"
	)

	const CurrentVersion = "1.0.0"

	var (
	homeDir, _      = os.UserHomeDir()
	baseDir         = filepath.Join(homeDir, ".gosip")
	binDir          = filepath.Join(baseDir, "bin")
	registryFile    = filepath.Join(baseDir, "registry.json")
	communityFile   = filepath.Join(baseDir, "community.json")
	stateFile       = filepath.Join(baseDir, "state.json")
	logDir          = filepath.Join(baseDir, "logs")
	sandboxDir      = filepath.Join(baseDir, "sandbox")
	backupBaseDir   = filepath.Join(baseDir, "backups")
	journalFile     = filepath.Join(logDir, "history")
	defaultRegistry = "https://raw.githubusercontent.com/Mkjmy/gosip-registry/main/registry.json"
	communityRegistry = "https://raw.githubusercontent.com/Mkjmy/gosip-registry/main/community.json"
	)

func main() {
	// Setup Cleanup on Interrupt (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		// Perform emergency cleanup
		os.RemoveAll(sandboxDir)
		// Try cleanup shadow dirs in /tmp (pattern based)
		matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "gosip-shadow-*"))
		for _, m := range matches {
			os.RemoveAll(m)
		}
		fmt.Print("\033[H\033[2J") // Clear screen
		os.Exit(0)
	}()

	if len(os.Args) < 2 {
		interactiveMenu()
		return
	}

	switch os.Args[1] {
	case "update": updateRegistry()
	case "init": initializeEnv()
	case "list": listInstalledApps()
	case "uninstall":
		if len(os.Args) < 3 {
			Red.Println(" [!] ERROR: Missing application name")
			return
		}
		uninstallApp(os.Args[2])
	default:
		Red.Printf(" [!] UNKNOWN_COMMAND: %s\n", os.Args[1])
	}
}

func interactiveMenu() {
	var allApps []registry.App
	data, err := os.ReadFile(registryFile)
	if err != nil {
		Red.Println(" [!] CRITICAL_ERROR: UNABLE_TO_READ_REGISTRY")
		return
	}

	var reg registry.Registry
	json.Unmarshal(data, &reg)
	for i := range reg.Apps {
		reg.Apps[i].IsOfficial = true
		allApps = append(allApps, reg.Apps[i])
	}

	// Load Community
	dataC, err := os.ReadFile(communityFile)
	if err == nil {
		var comm struct {
			Apps []registry.App `json:"apps"`
		}
		json.Unmarshal(dataC, &comm)
		for i := range comm.Apps {
			comm.Apps[i].IsOfficial = false
			allApps = append(allApps, comm.Apps[i])
		}
	}

	// Filter out non-existent repos
	var validApps []registry.App
	for _, app := range allApps {
		if registry.CheckRepoExists(app.Repo) {
			validApps = append(validApps, app)
		}
	}

	if len(validApps) == 0 {
		Red.Println(" [!] NO_VALID_APPS: All registry entries are unreachable.")
		return
	}

	for {
		choice, isShadow := customSelect(validApps)
		if isShadow {
			enterShadowMode(validApps[choice])
		} else {
			installApp(validApps[choice])
		}
	}
}
