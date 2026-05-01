package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gosip/internal/registry"
)

func initializeEnv() {
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(sandboxDir, 0755)
	os.MkdirAll(backupBaseDir, 0755)
	Green.Println(" [✓] Environment established at " + baseDir)
	pathLine := fmt.Sprintf("export PATH=\"$PATH:%s\"", binDir)
	shell := os.Getenv("SHELL")
	configFile := filepath.Join(homeDir, ".bashrc")
	if strings.Contains(shell, "zsh") {
		configFile = filepath.Join(homeDir, ".zshrc")
	}
	content, _ := os.ReadFile(configFile)
	if !strings.Contains(string(content), binDir) {
		f, _ := os.OpenFile(configFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		defer f.Close()
		f.WriteString("\n# GOSIP Bin Path\n" + pathLine + "\n")
		Green.Printf(" [+] PATH_CONFIGURED: %s updated.\n", configFile)
	} else {
		Blue.Println(" [SYS] PATH_ALREADY_CONFIGURED")
	}
}

func updateRegistry() {
	registry.SyncRegistry(defaultRegistry, communityRegistry, registryFile, communityFile, stateFile, CurrentVersion)
}

func selfUpdate(newVersion, repo string) {
	Yellow.Printf(" [!] SELF_UPDATE: v%s -> v%s\n", CurrentVersion, newVersion)
	executable, _ := os.Executable()
	tmpDir := filepath.Join(os.TempDir(), "gosip-build")
	os.RemoveAll(tmpDir)
	exec.Command("git", "clone", "--depth", "1", "https://github.com/"+repo+".git", tmpDir).Run()
	cmd := exec.Command("go", "build", "-o", "gosip-new", "main.go")
	cmd.Dir = tmpDir
	cmd.Run()
	os.Rename(filepath.Join(tmpDir, "gosip-new"), executable)
	Green.Println(" [+] REBUILT_AND_SWAPPED. PLEASE RESTART.")
	os.Exit(0)
}

func uninstallApp(appName string) {
	state, exists := registry.GetState(appName, stateFile)
	if !exists {
		Red.Printf("  [!] ERROR: %s is not recorded in system state.\n", appName)
		return
	}

	fmt.Print("\033[H\033[2J")
	printBanner()
	Yellow.Printf("\n [!] INITIATING_CLEANUP_PROTOCOL: %s\n", appName)
	
	// 1. Scan for backups in central repository
	backupPattern := filepath.Join(backupBaseDir, appName+".bak.*")
	backups, _ := filepath.Glob(backupPattern)
	
	if !customConfirm(" AUTHORIZE_REMOVAL_OF_CURRENT_UNIT?") {
		Yellow.Println("  [!] UNINSTALL_ABORTED.")
		return
	}

	// 2. Remove current installation
	if state.InstallPath != "" {
		Cyan.Printf("  [>] REMOVING_DATA: %s\n", state.InstallPath)
		os.RemoveAll(state.InstallPath)
	}
	if state.BinPath != "" {
		Cyan.Printf("  [>] REMOVING_SHORTCUT: %s\n", state.BinPath)
		os.Remove(state.BinPath)
	}

	// 3. Rollback Logic
	if len(backups) > 0 {
		fmt.Println()
		Blue.Printf("  [?] DETECTED %d BACKUP(S) IN CENTRAL_REPOSITORY.\n", len(backups))
		if customConfirm(" WOULD YOU LIKE TO ROLLBACK TO A PREVIOUS STATE?") {
			Yellow.Println("\n  SELECT_STRATEGY:")
			fmt.Println("  [1] RESTORE_LATEST (Most recent version)")
			fmt.Println("  [2] RESTORE_OLDEST (Original version)")
			
			fmt.Print("\n  Selection [1-2]: ")
			choice := readLine("")
			
			var targetBackup string
			if choice == "2" {
				targetBackup = backups[0] // Oldest
			} else {
				targetBackup = backups[len(backups)-1] // Latest
			}

			if targetBackup != "" {
				doneRoll, waitRoll := make(chan bool), make(chan bool)
				go registry.ShowDynamicProgress("RESTORING_UNIT", doneRoll, waitRoll)
				
				// Restore (Rename or copy)
				if err := os.Rename(targetBackup, state.InstallPath); err != nil {
					exec.Command("cp", "-r", targetBackup, state.InstallPath).Run()
				}
				
				doneRoll <- true
				<-waitRoll
				Green.Printf("  [✓] ROLLBACK_SUCCESSFUL: Restored from %s\n", filepath.Base(targetBackup))
			}
		}

		// 4. Purge remaining backups?
		remainingBackups, _ := filepath.Glob(backupPattern)
		if len(remainingBackups) > 0 {
			fmt.Println()
			if customConfirm(" PURGE_ALL_REMAINING_BACKUPS_FOR_THIS_APP?") {
				for _, b := range remainingBackups {
					os.RemoveAll(b)
				}
				Green.Println("  [✓] BACKUP_REPOSITORY_CLEANED.")
			}
		}
	}

	registry.RemoveState(appName, stateFile)
	Green.Printf("\n  [+] SYSTEM_PURGE_SUCCESSFUL: %s removed.\n", appName)
}

func listInstalledApps() {
	fmt.Print("\033[H\033[2J")
	printBanner()

	data, err := os.ReadFile(stateFile)
	if err != nil {
		Yellow.Println("\n [!] SYSTEM_STATUS: No applications installed yet.")
		return
	}

	var allStates map[string]registry.AppState
	json.Unmarshal(data, &allStates)

	if len(allStates) == 0 {
		Yellow.Println("\n [!] SYSTEM_STATUS: Registry is empty.")
		return
	}

	Purple.Println("\n ┌─ INSTALLED_APPLICATIONS ─────────────────────────────────┐")
	for name, state := range allStates {
		fmt.Printf(" │ %-12s %s (%s)\n", 
			Cyan.Sprint(name), 
			HiWhite+state.Version,
			Pink.Sprint(state.InstallDate))
		fmt.Printf(" │ %-12s %s\n", 
			Blue.Sprint("  PATH:"), 
			Yellow.Sprint(truncateString(state.InstallPath, 45)))
		fmt.Println(" ├──────────────────────────────────────────────────────────┤")
	}
	fmt.Print("\033[A\033[2K")
	Purple.Println(" └──────────────────────────────────────────────────────────┘")
	fmt.Printf("\n Total units deployed: %s\n", Green.Sprint(len(allStates)))
	waitReturn()
}

func checkDependencies(deps []string) bool {
	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			Red.Printf(" [!] MISSING_DEP: %s\n", dep); return false
		}
	}
	return true
}

func auditApp(app registry.App) {
	fmt.Println()
	Yellow.Println(" [!] AUDIT_MODE: INSPECTING_LOGIC")
	printAppReportDetailed(app, "AUDIT", true, "")
}

func installApp(app registry.App) {
	fmt.Print("\033[H\033[2J")
	printBanner()
	Pink.Printf("\n [>] INITIATING_STAGED_INSTALL: %s\n", app.Name)

	fmt.Println()
	tmpPath := filepath.Join(sandboxDir, app.Name)
	os.RemoveAll(tmpPath) 
	os.MkdirAll(sandboxDir, 0755)

	doneDL, waitDL := make(chan bool), make(chan bool)
	if app.Type == "git-config" {
		go registry.ShowDynamicProgress("INFILTRATING (CLONE)", doneDL, waitDL)
		exec.Command("git", "clone", "--quiet", "--depth", "1", "https://github.com/"+app.Repo+".git", tmpPath).Run()
		doneDL <- true
		<-waitDL
	} else {
		resp, err := http.Get(app.DownloadURL)
		if err != nil {
			Red.Println("  [!] DOWNLOAD_FAILED")
			return
		}
		defer resp.Body.Close()
		os.MkdirAll(tmpPath, 0755)
		binaryPath := filepath.Join(tmpPath, app.BinaryName)
		out, _ := os.Create(binaryPath)
		counter := &registry.WriteCounter{Total: resp.ContentLength, Label: "INFILTRATING (DL)"}
		io.Copy(out, io.TeeReader(resp.Body, counter))
		out.Close()
	}

	fmt.Println()
	doneScan, waitScan := make(chan bool), make(chan bool)
	go registry.ShowDynamicProgress("SECURITY_AUDITING", doneScan, waitScan)
	time.Sleep(1000 * time.Millisecond)
	var threats []string
	patterns := []string{"rm -rf /", "curl.*\\|.*sh", "wget.*\\|.*sh", "> /etc/", "chmod +x /"}
	for _, p := range patterns {
		cmd := exec.Command("grep", "-rInE", p, tmpPath)
		output, _ := cmd.CombinedOutput()
		if len(output) > 0 {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					threats = append(threats, strings.Replace(line, tmpPath+"/", "", 1))
				}
			}
		}
	}
	doneScan <- true
	<-waitScan

	if len(threats) > 0 {
		Red.Println("\n  [!] SECURITY_ALERT: SUSPICIOUS_LOGIC_IDENTIFIED")
		fmt.Println(Red.Sprint("  ┌─ THREAT_REPORT ──────────────────────────────────────────┐"))
		for i, t := range threats {
			if i >= 5 { fmt.Printf("  │ ... and %d more threats.                                 │\n", len(threats)-5); break }
			parts := strings.SplitN(t, ":", 3)
			if len(parts) == 3 {
				fileName := parts[0]
				if len(fileName) > 20 { fileName = "..." + fileName[len(fileName)-17:] }
				fmt.Printf("  │ %s:%-4s %-32s │\n", Pink.Sprint(fileName), Yellow.Sprint(parts[1]), Red.Sprint(truncateString(parts[2], 32)))
			}
		}
		fmt.Println(Red.Sprint("  └──────────────────────────────────────────────────────────┘"))
		if !customConfirm("POTENTIAL EXPLOIT FOUND. DISREGARD AND CONTINUE?") {
			os.RemoveAll(tmpPath); Yellow.Println("\n  [!] INSTALLATION_ABORTED: Security risk rejected."); waitReturn(); return
		}
	}

	targetPath := customInput("DEPLOYMENT_TARGET", registry.ExpandPath(app.TargetPath, homeDir))
	if !checkDependencies(app.Dependencies) {
		os.RemoveAll(tmpPath); Yellow.Println("\n  [!] INSTALLATION_ABORTED: Missing dependencies."); waitReturn(); return
	}

	if !customConfirm("AUTHORIZE_SYSTEM_DEPLOYMENT?") {
		os.RemoveAll(tmpPath); Yellow.Println("\n  [!] INSTALLATION_ABORTED: Deployment denied by user."); waitReturn(); return
	}

	var backupPath string
	if _, err := os.Stat(targetPath); err == nil {
		timestamp := time.Now().Format("20060102.150405")
		// Force default to central backup dir
		defaultBackup := filepath.Join(backupBaseDir, app.Name+".bak."+timestamp)
		
		fmt.Println()
		Yellow.Printf("  [!] UNIT_COLLISION: %s already exists.\n", targetPath)
		Cyan.Println("  [SYS] Old version will be archived in central repository.")
		
		backupPath = customInput("ARCHIVE_LOCATION", defaultBackup)
		clearLines(2) // Clear the collision and sys messages
		
		doneMove, waitMove := make(chan bool), make(chan bool)
		go registry.ShowDynamicProgress("ARCHIVING_OLD_UNIT", doneMove, waitMove)
		
		if err := registry.MoveToBackup(targetPath, backupPath); err != nil {
			doneMove <- true; <-waitMove
			Red.Printf("  [!] ARCHIVE_FAILED: %v\n", err)
			if !customConfirm("  PROCEED WITHOUT ARCHIVING? (Data may be merged/lost)") {
				os.RemoveAll(tmpPath); waitReturn(); return
			}
		} else {
			doneMove <- true; <-waitMove
		}
	}

	fmt.Println() 
	doneDep, waitDep := make(chan bool), make(chan bool)
	go registry.ShowDynamicProgress("DEPLOYING_UNITS", doneDep, waitDep)
	if app.Type == "git-config" {
		// Attempt rename (instant), fallback to cp+rm if needed
		if err := os.Rename(tmpPath, targetPath); err != nil {
			exec.Command("cp", "-r", tmpPath, targetPath).Run()
			os.RemoveAll(tmpPath)
		}
	} else {
		os.MkdirAll(binDir, 0755)
		finalBin := filepath.Join(binDir, app.BinaryName)
		if err := os.Rename(filepath.Join(tmpPath, app.BinaryName), finalBin); err != nil {
			exec.Command("cp", filepath.Join(tmpPath, app.BinaryName), finalBin).Run()
		}
		os.Chmod(finalBin, 0755)
	}
	doneDep <- true
	<-waitDep

	var finalBinPath string
	buildExecuted := false
	if app.PostInstall != "" {
		fmt.Println()
		if customConfirm("EXECUTE_PROVISIONING_LOGIC (BUILD)?") {
			fmt.Println(Blue.Sprint("  ┌─ PROVISIONING_MANIFEST ──────────────────────────────────┐"))
			lines := strings.Split(wrapText(app.PostInstall, 54), "\n")
			for _, line := range lines {
				fmt.Printf("  │ %-56s │\n", Yellow.Sprint(strings.TrimSpace(line)))
			}
			fmt.Println(Blue.Sprint("  └──────────────────────────────────────────────────────────┘"))

			if customConfirm("AUTHORIZE_EXECUTION?") {
				clearLines(len(lines) + 2) // Clear manifest box
				fmt.Println()
				doneBuild, waitBuild := make(chan bool), make(chan bool)
				go registry.ShowDynamicProgress("PROVISIONING_BUILD", doneBuild, waitBuild)

				cmd := exec.Command("sh", "-c", app.PostInstall)
				cmd.Dir = targetPath
				cmd.Run()

				doneBuild <- true
				<-waitBuild
				buildExecuted = true
			} else {
				clearLines(len(lines) + 3) // Clear manifest + initial question
			}
		}
	}

	if app.Type != "git-config" {
		if customConfirm("CREATE_GLOBAL_SHORTCUT?") {
			finalBinPath = filepath.Join(binDir, app.Name)
			script := fmt.Sprintf("#!/bin/bash\n%s \"$@\"", filepath.Join(binDir, app.BinaryName))
			os.WriteFile(finalBinPath, []byte(script), 0755)
		}
	}

	registry.SaveState(registry.AppState{Name: app.Name, Version: app.Version, InstallPath: targetPath, BinPath: finalBinPath}, stateFile)
	f, _ := os.OpenFile(journalFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		f.WriteString(fmt.Sprintf("[%s] SESSION_START\n", time.Now().Format("2006-01-02 15:04:05")))
		f.WriteString(fmt.Sprintf("  APP:      %s\n", app.Name)); f.WriteString(fmt.Sprintf("  VERSION:  %s\n", app.Version))
		f.WriteString(fmt.Sprintf("  PATH:     %s\n", targetPath))
		if backupPath != "" { f.WriteString(fmt.Sprintf("  BACKUP:   %s\n", backupPath)) } else { f.WriteString("  BACKUP:   None\n") }
		f.WriteString("---\n"); f.Close()
	}

	printAppReportDetailed(app, "INSTALLATION_SUCCESS", buildExecuted, backupPath)
	authorName := "Unknown Hacker"; parts := strings.Split(app.Repo, "/")
	if len(parts) > 0 { authorName = parts[0] }
	if app.AuthorNote != "" {
		fmt.Printf("\n  %s %s:\n  %s %s\n", Pink.Sprint("[AUTHOR_BROADCAST]"), HiWhite+authorName, Blue.Sprint("»"), Cyan.Sprint("\""+app.AuthorNote+"\""))
	} else {
		fmt.Printf("\n  %s %s: %s %s!\n", Pink.Sprint("[SYSTEM]"), HiWhite+authorName, Cyan.Sprint("Thanks for deploying"), Pink.Sprint(app.Name))
	}
	os.RemoveAll(tmpPath); waitReturn()
}

func enterShadowMode(app registry.App) {
	fmt.Print("\033[H\033[2J")
	printBanner()
	Purple.Printf("\n [!] INITIATING_SHADOW_MODE: %s\n", app.Name)
	Cyan.Println(" [SYS] Session is strictly ephemeral. Data resides in /tmp.")
	shadowPath := filepath.Join(os.TempDir(), "gosip-shadow-"+app.Name)
	os.RemoveAll(shadowPath); os.MkdirAll(shadowPath, 0755)

	doneDL, waitDL := make(chan bool), make(chan bool)
	if app.Type == "git-config" {
		go registry.ShowDynamicProgress("SHADOW_INFILTRATION", doneDL, waitDL)
		exec.Command("git", "clone", "--quiet", "--depth", "1", "https://github.com/"+app.Repo+".git", shadowPath).Run()
		doneDL <- true; <-waitDL
	} else {
		resp, _ := http.Get(app.DownloadURL); defer resp.Body.Close()
		binaryPath := filepath.Join(shadowPath, app.BinaryName); out, _ := os.Create(binaryPath)
		counter := &registry.WriteCounter{Total: resp.ContentLength, Label: "SHADOW_INFILTRATION"}
		io.Copy(out, io.TeeReader(resp.Body, counter)); out.Close(); os.Chmod(binaryPath, 0755)
	}

	if app.PostInstall != "" {
		fmt.Println(); doneBuild, waitBuild := make(chan bool), make(chan bool)
		go registry.ShowDynamicProgress("SHADOW_PROVISIONING", doneBuild, waitBuild)
		cmd := exec.Command("sh", "-c", app.PostInstall); cmd.Dir = shadowPath; cmd.Run()
		doneBuild <- true; <-waitBuild
	}

	fmt.Println(); Green.Printf(" [✓] SHADOW_SESSION_ACTIVE: %s is available.\n", app.Name)
	Yellow.Println(" [!] Type 'exit' to terminate. All data in /tmp will be purged.\n")
	shell := os.Getenv("SHELL"); if shell == "" { shell = "/bin/bash" }
	cmd := exec.Command(shell); env := os.Environ()
	newPath := "PATH=" + shadowPath + ":" + os.Getenv("PATH"); pathFound := false
	for i, e := range env { if strings.HasPrefix(e, "PATH=") { env[i] = newPath; pathFound = true; break } }
	if !pathFound { env = append(env, newPath) }
	cmd.Env = env; cmd.Stdin = os.Stdin; cmd.Stdout = os.Stdout; cmd.Stderr = os.Stderr; cmd.Run()
	os.RemoveAll(shadowPath); Green.Println("\n [✓] SHADOW_SESSION_TERMINATED: Temporary workspace purged."); time.Sleep(1 * time.Second)
}
