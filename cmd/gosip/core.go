package main

/*
 * GOSIP - CORE LOGIC
 * ------------------
 * File: cmd/gosip/core.go
 * Purpose: Contains the engine for installation, auditing, uninstallation, and shadow mode.
 */

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
	if err := os.MkdirAll(binDir, 0755); err != nil { Red.Printf(" [!] ERROR: Failed to create bin directory: %v\n", err) }
	if err := os.MkdirAll(logDir, 0755); err != nil { Red.Printf(" [!] ERROR: Failed to create logs directory: %v\n", err) }
	if err := os.MkdirAll(sandboxDir, 0755); err != nil { Red.Printf(" [!] ERROR: Failed to create sandbox directory: %v\n", err) }
	if err := os.MkdirAll(backupBaseDir, 0755); err != nil { Red.Printf(" [!] ERROR: Failed to create backups directory: %v\n", err) }
	
	Green.Println(" [✓] Environment established at " + baseDir)
	pathLine := fmt.Sprintf("export PATH=\"$PATH:%s\"", binDir); shell := os.Getenv("SHELL")
	configFile := filepath.Join(homeDir, ".bashrc"); if strings.Contains(shell, "zsh") { configFile = filepath.Join(homeDir, ".zshrc") }
	
	content, err := os.ReadFile(configFile)
	if err != nil && !os.IsNotExist(err) { 
		Red.Printf(" [!] ERROR: Failed to read shell config: %v\n", err)
		return
	}

	if !strings.Contains(string(content), binDir) {
		f, err := os.OpenFile(configFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil { Red.Printf(" [!] ERROR: Failed to open config for update: %v\n", err); return }
		defer f.Close()
		if _, err := f.WriteString("\n# GOSIP Bin Path\n" + pathLine + "\n"); err != nil {
			Red.Printf(" [!] ERROR: Failed to update shell path: %v\n", err)
		} else {
			Green.Printf(" [+] PATH_CONFIGURED: %s updated.\n", configFile)
		}
	} else { Blue.Println(" [SYS] PATH_ALREADY_CONFIGURED") }
}

func selfUpdate(newVersion, repo string) {
	Yellow.Printf(" [!] SELF_UPDATE: v%s -> v%s\n", CurrentVersion, newVersion)
	executable, err := os.Executable()
	if err != nil { Red.Printf(" [!] ERROR: Unable to locate binary: %v\n", err); return }
	
	tmpDir := filepath.Join(os.TempDir(), "gosip-build")
	os.RemoveAll(tmpDir)
	
	if err := exec.Command("git", "clone", "--depth", "1", "https://github.com/"+repo+".git", tmpDir).Run(); err != nil {
		Red.Printf(" [!] ERROR: Failed to clone source: %v\n", err); return
	}
	
	cmd := exec.Command("go", "build", "-o", "gosip-new", "main.go")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		Red.Printf(" [!] ERROR: Build failed: %v\n", err); return
	}
	
	if err := os.Rename(filepath.Join(tmpDir, "gosip-new"), executable); err != nil {
		Red.Printf(" [!] ERROR: Failed to swap binaries: %v\n", err); return
	}
	
	Green.Println(" [+] REBUILT_AND_SWAPPED. PLEASE RESTART."); os.Exit(0)
}

func uninstallApp(appName string) {
	state, exists := registry.GetState(appName, stateFile)
	if !exists { Red.Printf("  [!] ERROR: %s not found.\n", appName); waitReturn(); return }
	
	Yellow.Printf("\n [!] INITIATING_CLEANUP_PROTOCOL: %s\n", appName)
	backupPattern := filepath.Join(backupBaseDir, appName+".bak.*")
	backups, _ := filepath.Glob(backupPattern)
	
	fmt.Println()
	if !customConfirm(" AUTHORIZE_REMOVAL?") { Yellow.Println("  [!] ABORTED."); waitReturn(); return }
	
	if state.InstallPath != "" { 
		if err := os.RemoveAll(state.InstallPath); err != nil { Red.Printf(" [!] ERROR: Failed to remove files: %v\n", err) }
	}
	if state.BinPath != "" { 
		if err := os.Remove(state.BinPath); err != nil { Red.Printf(" [!] ERROR: Failed to remove link: %v\n", err) }
	}

	if len(backups) > 0 {
		fmt.Println(); Blue.Printf("  [?] DETECTED %d BACKUPS.\n", len(backups))
		fmt.Println()
		if customConfirm(" ROLLBACK?") {
			fmt.Print("\n  Selection [1-Latest, 2-Oldest]: "); choice := readKey()
			target := backups[len(backups)-1]; if choice == "2" { target = backups[0] }
			fmt.Println(); doneRoll, waitRoll := registry.ShowDynamicProgress("RESTORING_UNIT")
			if err := os.Rename(target, state.InstallPath); err != nil { 
				if err := exec.Command("cp", "-r", target, state.InstallPath).Run(); err != nil {
					Red.Printf(" [!] ERROR: Rollback failed: %v\n", err)
				}
			}
			doneRoll <- true; <-waitRoll
		}
		fmt.Println()
		if customConfirm(" PURGE BACKUPS?") { 
			for _, b := range backups { 
				if err := os.RemoveAll(b); err != nil { Red.Printf(" [!] ERROR: Failed to purge %s: %v\n", b, err) }
			} 
		}
	}
	registry.RemoveState(appName, stateFile); Green.Printf("\n  [+] REMOVED: %s\n", appName); waitReturn()
}

func dumpApps(path string) {
	data, err := os.ReadFile(stateFile)
	if err != nil { Red.Printf(" [!] ERROR: Failed to read state: %v\n", err); return }
	if err := os.WriteFile(path, data, 0644); err != nil {
		Red.Printf(" [!] ERROR: Failed to write dump: %v\n", err); return
	}
	Green.Printf(" [+] SNAPSHOT_CREATED: %s\n", path)
}

func restoreApps(path string) {
	data, err := os.ReadFile(path)
	if err != nil { Red.Printf(" [!] ERROR: Snapshot not found: %s\n", path); return }
	
	var allStates map[string]registry.AppState
	if err := json.Unmarshal(data, &allStates); err != nil {
		Red.Printf(" [!] ERROR: Invalid snapshot format: %v\n", err); return
	}

	fmt.Printf("\n [>] RESTORING FROM: %s\n", path)
	for _, state := range allStates { 
		if app, found := findApp(state.Name); found { 
			installApp(app) 
		} else {
			Yellow.Printf(" [!] WARNING: App '%s' not found in registries. Skipping.\n", state.Name)
		}
	}
	Green.Println("\n [✓] RESTORE COMPLETE.")
}

func getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error { if err == nil && !info.IsDir() { size += info.Size() }; return nil }); return size
}

func formatSize(bytes int64) string {
	const unit = 1024; if bytes < unit { return fmt.Sprintf("%d B", bytes) }; div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit { div *= unit; exp++ }; return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func listInstalledApps() {
	data, err := os.ReadFile(stateFile); if err != nil { Yellow.Println("\n [!] No apps installed."); waitReturn(); return }
	var allStates map[string]registry.AppState; json.Unmarshal(data, &allStates)
	if len(allStates) == 0 { Yellow.Println("\n [!] Registry empty."); waitReturn(); return }
	fmt.Println(); boxWidth := 63; drawBoxBorder(Purple, "SYSTEM_DASHBOARD", "┏", boxWidth)
	var totalSize int64
	for name, state := range allStates {
		size := getDirSize(state.InstallPath); totalSize += size; status := Green.Sprint("[HEALTHY]"); if _, err := os.Stat(state.InstallPath); err != nil { status = Red.Sprint("[BROKEN]") }
		identity := fmt.Sprintf("%s %s", HiWhite+name, Pink.Sprint("("+state.Version+")"))
		printBoxLine(Purple, "UNIT", identity, boxWidth); printBoxLine(Purple, "STAT", fmt.Sprintf("%s | SIZE: %s", status, formatSize(size)), boxWidth); printBoxWrapped(Purple, "PATH", state.InstallPath, boxWidth)
		fmt.Printf("  " + Purple.Sprint("┠") + strings.Repeat(Purple.Sprint("━"), boxWidth-4) + Purple.Sprint("┨") + "\n")
	}
	fmt.Print("\033[A\033[2K"); drawBoxBorder(Purple, "", "┗", boxWidth)
	fmt.Printf("\n Total units: %s | Disk usage: %s\n", Green.Sprint(len(allStates)), Yellow.Sprint(formatSize(totalSize))); waitReturn()
}

func checkDependencies(deps []string) bool {
	for _, dep := range deps { if _, err := exec.LookPath(dep); err != nil { Red.Printf(" [!] MISSING: %s\n", dep); return false } }; return true
}

func auditApp(app registry.App) {
	fmt.Println(); Yellow.Println(" [!] AUDIT_MODE: INSPECTING_LOGIC")
	printAppReportDetailed(app, "AUDIT", true, "", nil); waitReturn()
}

func checkBinaryBlobs(dir string) []string {
	var blobs []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && !strings.HasPrefix(filepath.Base(path), ".") {
			ext := strings.ToLower(filepath.Ext(path)); binaryExts := map[string]bool{".exe": true, ".so": true, ".dll": true, ".bin": true}
			if binaryExts[ext] { blobs = append(blobs, path) } else {
				out, _ := exec.Command("file", path).Output()
				if (strings.Contains(string(out), "executable") || strings.Contains(string(out), "shared object")) && !strings.Contains(string(out), "script") { blobs = append(blobs, path) }
			}
		}
		return nil
	}); return blobs
}

func detectBuildSystem(dir string) string {
	systems := map[string]string{ "Makefile": "Make", "go.mod": "Go", "package.json": "Node.js", "Cargo.toml": "Rust" }
	for file, name := range systems { if _, err := os.Stat(filepath.Join(dir, file)); err == nil { return name } }; return "None Detected"
}

func checkLicense(dir string) string {
	files, _ := os.ReadDir(dir)
	for _, f := range files { n := strings.ToUpper(f.Name()); if strings.Contains(n, "LICENSE") || strings.Contains(n, "COPYING") { return f.Name() } }; return "Missing"
}

func installApp(app registry.App) {
	fmt.Print("\033[H\033[2J") // FULL CLEAR
	Pink.Printf("\n [>] INITIATING_STAGED_INSTALL: %s\n", app.Name)
	tmpPath := filepath.Join(sandboxDir, app.Name)
	if err := os.RemoveAll(tmpPath); err != nil { Red.Printf(" [!] WARNING: Cleanup failed: %v\n", err) }
	if err := os.MkdirAll(sandboxDir, 0755); err != nil { Red.Printf(" [!] ERROR: Sandbox creation failed: %v\n", err); waitReturn(); return }

	// --- 1. INFILTRATING ---
	fmt.Println()
	if app.Type == "git-config" {
		doneDL, waitDL := registry.ShowDynamicProgress("INFILTRATING (CLONE)")
		if app.CommitHash != "" {
			if err := exec.Command("git", "clone", "--quiet", "https://github.com/"+app.Repo+".git", tmpPath).Run(); err != nil { Red.Printf(" [!] CLONE_FAILED: %v\n", err); waitReturn(); return }
			if err := exec.Command("git", "-C", tmpPath, "checkout", "--quiet", app.CommitHash).Run(); err != nil { Red.Printf(" [!] CHECKOUT_FAILED: %v\n", err); waitReturn(); return }
		} else {
			if err := exec.Command("git", "clone", "--quiet", "--depth", "1", "https://github.com/"+app.Repo+".git", tmpPath).Run(); err != nil { Red.Printf(" [!] CLONE_FAILED: %v\n", err); waitReturn(); return }
		}
		os.RemoveAll(filepath.Join(tmpPath, ".git")); doneDL <- true; <-waitDL
	} else {
		resp, err := http.Get(app.DownloadURL)
		if err != nil { Red.Printf(" [!] DOWNLOAD_FAILED: %v\n", err); waitReturn(); return }
		defer resp.Body.Close()
		if resp.StatusCode != 200 { Red.Printf(" [!] DOWNLOAD_FAILED: Status %d\n", resp.StatusCode); waitReturn(); return }
		
		if err := os.MkdirAll(tmpPath, 0755); err != nil { Red.Printf(" [!] DIR_FAILED: %v\n", err); waitReturn(); return }
		binP := filepath.Join(tmpPath, app.BinaryName)
		out, err := os.Create(binP)
		if err != nil { Red.Printf(" [!] CREATE_FAILED: %v\n", err); waitReturn(); return }
		
		doneDL, waitDL := registry.ShowDynamicProgress("INFILTRATING (DL)")
		if _, err := io.Copy(out, io.TeeReader(resp.Body, &registry.WriteCounter{Total: resp.ContentLength, Label: "INFILTRATING (DL)"})); err != nil {
			Red.Printf(" [!] STREAM_FAILED: %v\n", err)
			out.Close(); waitReturn(); return
		}
		out.Close(); doneDL <- true; <-waitDL 
	}
	fmt.Println() // PERSISTENT GAP

	// --- 2. AUDITING ---
	doneAudit, waitAudit := registry.ShowDynamicProgress("DEEP_AUDITING")
	var threats []string; patterns := []string{"rm -rf /", "curl.*\\|.*sh", "sudo ", "crontab "}
	for _, p := range patterns { 
		cmd := exec.Command("grep", "-rInE", p, tmpPath)
		out, _ := cmd.CombinedOutput() 
		if len(out) > 0 { 
			for _, l := range strings.Split(string(out), "\n") { 
				if strings.TrimSpace(l) != "" { threats = append(threats, strings.Replace(l, tmpPath+"/", "", 1)) } 
			} 
		} 
	}
	blobs := checkBinaryBlobs(tmpPath); build := detectBuildSystem(tmpPath); lic := checkLicense(tmpPath); time.Sleep(500 * time.Millisecond); doneAudit <- true; <-waitAudit
	fmt.Println() // PERSISTENT GAP

	// --- AUDIT REPORT (Temporary) ---
	auditColor := Cyan; if len(threats) > 0 || len(blobs) > 0 { auditColor = Red } else if lic == "Missing" { auditColor = Yellow }
	drawBoxBorder(auditColor, "DEEP_AUDIT_REPORT", "┏", 63)
	sLogic := Green.Sprint("[✓] CLEAN"); if len(threats) > 0 { sLogic = Red.Sprint("[!] WARNING") }
	sIntegrity := Green.Sprint("[✓] SOURCE_ONLY"); if len(blobs) > 0 { sIntegrity = Yellow.Sprint("[!] BLOBS FOUND") }
	trustStat := Green.Sprint("[✓] PINNED (Safe)"); if app.CommitHash == "" { trustStat = Yellow.Sprint("[!] UNPINNED") }
	printBoxLine(auditColor, "LOGIC", sLogic, 63); printBoxLine(auditColor, "INTEGRITY", sIntegrity, 63); printBoxLine(auditColor, "TRUST", trustStat, 63); printBoxLine(auditColor, "BUILD", Blue.Sprint(build), 63); printBoxLine(auditColor, "LICENSE", lic, 63); drawBoxBorder(auditColor, "", "┗", 63)
	cont := customConfirm("CONTINUE?")
	clearLines(8) // Box(7) + Confirm(1)
	if !cont { os.RemoveAll(tmpPath); return }

	// --- TARGET (Temporary) ---
	target := flagTarget
	if target == "" { 
		target = customInput("TARGET", registry.ExpandPath(app.TargetPath, homeDir))
		clearLines(1) 
	}

	// --- AUTHORIZE (Temporary) ---
	depsOk := checkDependencies(app.Dependencies)
	auth := customConfirm("AUTHORIZE DEPLOYMENT?")
	clearLines(1) 
	if !depsOk || !auth { os.RemoveAll(tmpPath); return }

	// --- ARCHIVING ---
	if _, err := os.Stat(target); err == nil {
		bak := filepath.Join(backupBaseDir, app.Name+".bak."+time.Now().Format("20060102.150405"))
		if !flagAuto {
			Yellow.Printf("  [!] EXISTS: %s\n", target)
			bak = customInput("  BACKUP_TO", bak)
			clearLines(2) 
		}
		doneM, waitM := registry.ShowDynamicProgress("ARCHIVING")
		if err := registry.MoveToBackup(target, bak); err != nil { Red.Printf(" [!] BACKUP_FAILED: %v\n", err) }
		doneM <- true; <-waitM
		fmt.Println() // PERSISTENT GAP
	}

	// --- DEPLOYING ---
	doneD, waitD := registry.ShowDynamicProgress("DEPLOYING")
	if app.Type == "git-config" {
		if err := os.Rename(tmpPath, target); err != nil { 
			if err := exec.Command("cp", "-r", tmpPath, target).Run(); err != nil {
				Red.Printf(" [!] DEPLOY_FAILED: %v\n", err); waitReturn(); return
			}
			os.RemoveAll(tmpPath)
		}
	} else {
		if err := os.MkdirAll(binDir, 0755); err != nil { Red.Printf(" [!] BIN_DIR_FAILED: %v\n", err) }
		fBin := filepath.Join(binDir, app.BinaryName)
		if err := os.Rename(filepath.Join(tmpPath, app.BinaryName), fBin); err != nil { Red.Printf(" [!] BIN_MOVE_FAILED: %v\n", err); waitReturn(); return }
		if err := os.Chmod(fBin, 0755); err != nil { Red.Printf(" [!] CHMOD_FAILED: %v\n", err) }
	}
	doneD <- true; <-waitD
	fmt.Println() // PERSISTENT GAP

	// --- BUILD (Temporary) ---
	buildDone := false
	if app.PostInstall != "" {
		runB := customConfirm("RUN BUILD?")
		clearLines(1)
		if runB {
			if !flagAuto && !flagExec {
				fmt.Println(); drawBoxBorder(Blue, "BUILD_MANIFEST", "┏", 63); printBoxWrapped(Blue, "LOGIC", app.PostInstall, 63); drawBoxBorder(Blue, "", "┗", 63)
				authB := customConfirm("AUTHORIZE?")
				clearLines(6) 
				if authB {
					doneB, waitB := registry.ShowDynamicProgress("BUILDING")
					cmd := exec.Command("sh", "-c", app.PostInstall)
					cmd.Dir = target
					if err := cmd.Run(); err != nil { Red.Printf(" [!] BUILD_FAILED: %v\n", err) } else { buildDone = true }
					doneB <- true; <-waitB
					fmt.Println() // Persistent Gap
				}
			} else {
				doneB, waitB := registry.ShowDynamicProgress("BUILDING")
				cmd := exec.Command("sh", "-c", app.PostInstall)
				cmd.Dir = target
				if err := cmd.Run(); err != nil { Red.Printf(" [!] BUILD_FAILED: %v\n", err) } else { buildDone = true }
				doneB <- true; <-waitB
				fmt.Println() // Persistent Gap
			}
		}
	}

	// --- SHORTCUT (Temporary) ---
	shouldShortcut := flagAuto || flagYes
	if !shouldShortcut && app.Type != "git-config" {
		shortcut := customConfirm("SHORTCUT?")
		clearLines(1)
		shouldShortcut = shortcut
	}

	if shouldShortcut && app.Type != "git-config" {
		scPath := filepath.Join(binDir, app.Name)
		if err := os.WriteFile(scPath, []byte("#!/bin/bash\n"+filepath.Join(binDir, app.BinaryName)+" \"$@\""), 0755); err != nil {
			Red.Printf(" [!] SHORTCUT_FAILED: %v\n", err)
		}
	}

	registry.SaveState(registry.AppState{Name: app.Name, Version: app.Version, InstallPath: target}, stateFile)
	printAppReportDetailed(app, "SUCCESS", buildDone, "", &AuditSummary{Logic: sLogic, Integrity: sIntegrity, Build: Blue.Sprint(detectBuildSystem(target)), License: lic})
	os.RemoveAll(tmpPath); waitReturn()
}

func enterShadowMode(app registry.App) {
	fmt.Print("\033[H\033[2J")
	Purple.Printf("\n [!] INITIATING_SHADOW_MODE: %s\n", app.Name)
	Cyan.Println(" [SYS] Session is strictly ephemeral. Data resides in /tmp.")
	shadowPath := filepath.Join(os.TempDir(), "gosip-shadow-"+app.Name)
	os.RemoveAll(shadowPath)
	if err := os.MkdirAll(shadowPath, 0755); err != nil { Red.Printf(" [!] ERROR: %v\n", err); waitReturn(); return }

	if app.Type == "git-config" {
		doneDL, waitDL := registry.ShowDynamicProgress("SHADOW_INFILTRATION")
		if app.CommitHash != "" { 
			if err := exec.Command("git", "clone", "--quiet", "https://github.com/"+app.Repo+".git", shadowPath).Run(); err != nil { Red.Printf(" [!] CLONE_FAILED: %v\n", err); waitReturn(); return }
			if err := exec.Command("git", "-C", shadowPath, "checkout", "--quiet", app.CommitHash).Run(); err != nil { Red.Printf(" [!] CHECKOUT_FAILED: %v\n", err); waitReturn(); return }
		} else { 
			if err := exec.Command("git", "clone", "--quiet", "--depth", "1", "https://github.com/"+app.Repo+".git", shadowPath).Run(); err != nil { Red.Printf(" [!] CLONE_FAILED: %v\n", err); waitReturn(); return }
		}
		os.RemoveAll(filepath.Join(shadowPath, ".git")); doneDL <- true; <-waitDL
	} else {
		resp, err := http.Get(app.DownloadURL)
		if err != nil { Red.Printf(" [!] DOWNLOAD_FAILED: %v\n", err); waitReturn(); return }
		defer resp.Body.Close()
		
		binaryPath := filepath.Join(shadowPath, app.BinaryName)
		out, err := os.Create(binaryPath)
		if err != nil { Red.Printf(" [!] CREATE_FAILED: %v\n", err); waitReturn(); return }
		
		doneDL, waitDL := registry.ShowDynamicProgress("SHADOW_INFILTRATION")
		if _, err := io.Copy(out, io.TeeReader(resp.Body, &registry.WriteCounter{Total: resp.ContentLength, Label: "SHADOW_INFILTRATION"})); err != nil { Red.Printf(" [!] STREAM_FAILED: %v\n", err) }
		out.Close(); os.Chmod(binaryPath, 0755); doneDL <- true; <-waitDL
	}

	if app.PostInstall != "" {
		fmt.Println(); doneBuild, waitBuild := registry.ShowDynamicProgress("SHADOW_PROVISIONING")
		cmd := exec.Command("sh", "-c", app.PostInstall)
		cmd.Dir = shadowPath
		if err := cmd.Run(); err != nil { Red.Printf(" [!] SHADOW_BUILD_FAILED: %v\n", err) }
		doneBuild <- true; <-waitBuild
	}

	fmt.Println(); Green.Printf(" [✓] SHADOW_SESSION_ACTIVE: %s is available.\n", app.Name)
	Yellow.Println(" [!] Type 'exit' to terminate. All data in /tmp will be purged.\n")
	shell := os.Getenv("SHELL"); if shell == "" { shell = "/bin/bash" }
	cmd := exec.Command(shell); env := os.Environ()
	newPath := "PATH=" + shadowPath + ":" + os.Getenv("PATH")
	pathFound := false
	for i, e := range env { if strings.HasPrefix(e, "PATH=") { env[i] = newPath; pathFound = true; break } }
	if !pathFound { env = append(env, newPath) }
	cmd.Env = env; cmd.Stdin = os.Stdin; cmd.Stdout = os.Stdout; cmd.Stderr = os.Stderr; cmd.Run()
	os.RemoveAll(shadowPath); Green.Println("\n [✓] SHADOW_SESSION_TERMINATED: Temporary workspace purged."); waitReturn()
}
