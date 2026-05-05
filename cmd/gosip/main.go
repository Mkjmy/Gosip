package main

/*
 * GOSIP - MAIN ENTRY POINT
 * -----------------------
 * File: cmd/gosip/main.go
 * Purpose: Handles CLI routing, flag parsing, and global configuration.
 *
 * Sections:
 * - [1-40]: Imports, Global Constants, and Path Variables
 * - [41-75]: Registry & Trust Persistence Logic
 * - [77-98]: CLI Flag Parsing
 * - [100-160]: Main Execution Loop & Signal Handling
 * - [162-250]: Command Handlers (Search, Registry, Trust, etc.)
 */

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	sourcesFile     = filepath.Join(baseDir, "sources.json")
	trustedFile     = filepath.Join(baseDir, "trusted_authors.json")
	defaultRegistry = "https://raw.githubusercontent.com/Mkjmy/gosip-registry/main/registry.json"
	communityRegistry = "https://raw.githubusercontent.com/Mkjmy/gosip-registry/main/community.json"

	flagAuto   bool
	flagYes    bool
	flagExec   bool
	flagTarget string
	flagCLI    bool
)

func loadSources() []registry.RegistrySource {
	var sources []registry.RegistrySource
	data, err := os.ReadFile(sourcesFile)
	if err != nil {
		sources = []registry.RegistrySource{
			{Name: "official", URL: defaultRegistry, File: "registry.json"},
			{Name: "community", URL: communityRegistry, File: "community.json"},
		}
		saveSources(sources)
		return sources
	}
	if err := json.Unmarshal(data, &sources); err != nil {
		Red.Printf(" [!] CRITICAL_ERROR: Unable to parse sources.json: %v\n", err)
		return []registry.RegistrySource{}
	}
	return sources
}

func saveSources(sources []registry.RegistrySource) {
	data, err := json.MarshalIndent(sources, "", "  ")
	if err != nil {
		Red.Printf(" [!] ERROR: Failed to marshal sources: %v\n", err)
		return
	}
	if err := os.WriteFile(sourcesFile, data, 0644); err != nil {
		Red.Printf(" [!] ERROR: Failed to write sources file: %v\n", err)
	}
}

func loadTrustedAuthors() []string {
	var authors []string
	data, err := os.ReadFile(trustedFile)
	if err != nil {
		if os.IsNotExist(err) { return []string{} }
		Red.Printf(" [!] ERROR: Failed to read trusted_authors.json: %v\n", err)
		return []string{}
	}
	if err := json.Unmarshal(data, &authors); err != nil {
		Red.Printf(" [!] ERROR: Failed to parse trusted_authors.json: %v\n", err)
		return []string{}
	}
	return authors
}

func saveTrustedAuthors(authors []string) {
	data, err := json.MarshalIndent(authors, "", "  ")
	if err != nil {
		Red.Printf(" [!] ERROR: Failed to marshal trusted authors: %v\n", err)
		return
	}
	if err := os.WriteFile(trustedFile, data, 0644); err != nil {
		Red.Printf(" [!] ERROR: Failed to write trusted_authors.json: %v\n", err)
	}
}

func isAuthorTrusted(author string) bool {
	trusted := loadTrustedAuthors()
	for _, t := range trusted {
		if strings.EqualFold(t, author) { return true }
	}
	return false
}

func parseFlags() []string {
	var remaining []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-a" || arg == "--auto" {
			flagAuto = true; flagYes = true; flagExec = true
		} else if arg == "-y" || arg == "--yes" {
			flagYes = true
		} else if arg == "-e" || arg == "--exec" {
			flagExec = true
		} else if (arg == "-t" || arg == "--target") && i+1 < len(os.Args) {
			flagTarget = os.Args[i+1]; i++
		} else if strings.HasPrefix(arg, "--target=") {
			flagTarget = strings.SplitN(arg, "=", 2)[1]
		} else {
			remaining = append(remaining, arg)
		}
	}
	return remaining
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		os.RemoveAll(sandboxDir)
		matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "gosip-shadow-*"))
		for _, m := range matches { os.RemoveAll(m) }
		fmt.Print("\033[?1049l") // Emergency exit
		os.Exit(0)
	}()

	args := parseFlags()
	if len(args) < 1 {
		enterAltScreen()
		defer exitAltScreen()
		interactiveMenu()
		return
	}
	
	flagCLI = true
	switch args[0] {
	case "update": updateRegistry()
	case "init": initializeEnv()
	case "list": 
		enterAltScreen(); defer exitAltScreen()
		listInstalledApps()
	case "registry": manageRegistries(args[1:])
	case "trust": manageTrust(args[1:])
	case "dump":
		path := "gosip-snapshot.json"
		if len(args) > 1 { path = args[1] }
		dumpApps(path)
	case "restore":
		if len(args) < 2 { Red.Println(" [!] Usage: gosip restore <file>"); return }
		restoreApps(args[1])
	case "check":
		if len(args) < 2 { Red.Println(" [!] ERROR: Missing application name"); return }
		checkApp(args[1])
	case "install":
		if len(args) < 2 { Red.Println(" [!] ERROR: Missing application name"); return }
		app, found := findApp(args[1])
		if !found { Red.Printf(" [!] ERROR: App '%s' not found\n", args[1]); return }
		enterAltScreen(); defer exitAltScreen()
		installApp(app)
	case "shadow":
		if len(args) < 2 { Red.Println(" [!] ERROR: Missing application name"); return }
		app, found := findApp(args[1])
		if !found { Red.Printf(" [!] ERROR: App '%s' not found\n", args[1]); return }
		enterAltScreen(); defer exitAltScreen()
		enterShadowMode(app)
	case "audit":
		if len(args) < 2 { Red.Println(" [!] ERROR: Missing application name"); return }
		app, found := findApp(args[1])
		if !found { Red.Printf(" [!] ERROR: App '%s' not found\n", args[1]); return }
		enterAltScreen(); defer exitAltScreen()
		auditApp(app)
	case "info":
		if len(args) < 2 { Red.Println(" [!] ERROR: Missing application name"); return }
		app, found := findApp(args[1])
		if !found { Red.Printf(" [!] ERROR: App '%s' not found\n", args[1]); return }
		enterAltScreen(); defer exitAltScreen()
		printAppReportDetailed(app, "INFO_MANIFEST", false, "", nil)
		waitReturn()
	case "search":
		if len(args) < 2 { Red.Println(" [!] ERROR: Missing search query"); return }
		searchApps(args[1])
	default:
		Red.Printf(" [!] UNKNOWN_COMMAND: %s\n", args[0])
	}
}

func updateRegistry() {
	sources := loadSources()
	registry.SyncRegistry(sources, baseDir, stateFile, CurrentVersion)
}

func loadAllApps() []registry.App {
	sources := loadSources()
	var allApps []registry.App
	for _, src := range sources {
		data, err := os.ReadFile(filepath.Join(baseDir, src.File))
		if err != nil { continue }
		var reg registry.Registry
		json.Unmarshal(data, &reg)
		for i := range reg.Apps {
			reg.Apps[i].IsOfficial = (src.Name == "official")
			allApps = append(allApps, reg.Apps[i])
		}
	}
	return allApps
}

func findApp(name string) (registry.App, bool) {
	apps := loadAllApps()
	for _, a := range apps {
		if a.Name == name { return a, true }
	}
	return registry.App{}, false
}

func manageRegistries(args []string) {
	sources := loadSources()
	if len(args) < 1 {
		fmt.Printf("\n  %s:\n", Cyan.Sprint("CONFIGURED_REGISTRIES"))
		for _, s := range sources { fmt.Printf("  - %-12s %s\n", HiWhite+s.Name, Yellow.Sprint(s.URL)) }
		return
	}
	switch args[0] {
	case "add":
		if len(args) < 3 { Red.Println(" [!] Usage: gosip registry add <name> <url>"); return }
		name, url := args[1], args[2]
		sources = append(sources, registry.RegistrySource{Name: name, URL: url, File: name + ".json"})
		saveSources(sources); Green.Printf(" [+] Registry '%s' added.\n", name)
	case "remove":
		if len(args) < 2 { Red.Println(" [!] Usage: gosip registry remove <name>"); return }
		var newSources []registry.RegistrySource
		for _, s := range sources { if s.Name != args[1] { newSources = append(newSources, s) } }
		saveSources(newSources); Green.Printf(" [-] Registry '%s' removed.\n", args[1])
	}
}

func manageTrust(args []string) {
	trusted := loadTrustedAuthors()
	if len(args) < 1 {
		fmt.Printf("\n  %s:\n", Purple.Sprint("TRUSTED_AUTHORS_LIST"))
		if len(trusted) == 0 { fmt.Println("  (No authors trusted yet)") }
		for _, a := range trusted { fmt.Printf("  [★] %s\n", HiWhite+a) }
		return
	}
	switch args[0] {
	case "add":
		if len(args) < 2 { Red.Println(" [!] Usage: gosip trust add <username>"); return }
		name := args[1]
		if isAuthorTrusted(name) { Yellow.Printf(" [!] %s is already trusted.\n", name); return }
		trusted = append(trusted, name); saveTrustedAuthors(trusted)
		Green.Printf(" [★] %s is now a TRUSTED_DEVELOPER.\n", name)
	case "remove":
		if len(args) < 2 { Red.Println(" [!] Usage: gosip trust remove <username>"); return }
		var newList []string
		for _, a := range trusted { if !strings.EqualFold(a, args[1]) { newList = append(newList, a) } }
		saveTrustedAuthors(newList); Green.Printf(" [-] Removed %s from trusted list.\n", args[1])
	}
}

func searchApps(query string) {
	apps := loadAllApps(); query = strings.ToLower(query); found := false
	fmt.Printf("\n  %s '%s':\n", Cyan.Sprint("SEARCH_RESULTS_FOR"), query)
	fmt.Println("  " + strings.Repeat("─", 50))
	for _, a := range apps {
		if strings.Contains(strings.ToLower(a.Name), query) || strings.Contains(strings.ToLower(a.Description), query) {
			status := Blue.Sprint("[○] AVAILABLE")
			if _, exists := registry.GetState(a.Name, stateFile); exists { status = Green.Sprint("[●] INSTALLED") }
			fmt.Printf("  %-20s %-10s %s\n", HiWhite+a.Name, Pink.Sprint(a.Version), status)
			fmt.Printf("    \033[2m%s\033[0m\n\n", truncateString(a.Description, 60))
			found = true
		}
	}
	if !found { Red.Println("    [!] No applications match your query.") }
}

func checkApp(name string) {
	app, found := findApp(name)
	if !found { Red.Printf(" [!] ERROR: Application '%s' not found\n", name); return }
	state, installed := registry.GetState(name, stateFile)
	fmt.Printf("\n  %s: %s\n", Cyan.Sprint("SYSTEM_CHECK_FOR"), HiWhite+name)
	fmt.Println("  " + strings.Repeat("─", 40))
	if installed {
		status := Green.Sprint("[●] INSTALLED")
		if state.Version != app.Version { status = Yellow.Sprint("[!] UPDATE_AVAILABLE") }
		fmt.Printf("  STATUS:  %s\n  VERSION: %s (Registry: %s)\n  PATH:    %s\n  DATE:    %s\n", status, Green.Sprint(state.Version), Pink.Sprint(app.Version), Yellow.Sprint(state.InstallPath), state.InstallDate)
	} else {
		fmt.Printf("  STATUS:  %s\n  VERSION: %s (Latest)\n  DESC:    %s\n", Blue.Sprint("[○] NOT_INSTALLED"), Pink.Sprint(app.Version), app.Description)
	}
}

func interactiveMenu() {
	allApps := loadAllApps()
	if len(allApps) == 0 { Red.Println(" [!] CRITICAL_ERROR: UNABLE_TO_READ_REGISTRY"); return }
	var validApps []registry.App
	for _, app := range allApps { if registry.CheckRepoExists(app.Repo) { validApps = append(validApps, app) } }
	if len(validApps) == 0 { Red.Println(" [!] NO_VALID_APPS: All registry entries are unreachable."); return }
	customSelect(validApps)
}
