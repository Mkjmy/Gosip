package main

/*
 * GOSIP - USER INTERFACE
 * ----------------------
 * File: cmd/gosip/ui.go
 * Purpose: Manages the TUI, terminal manipulation, colors, and interactive menus.
 */

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gosip/internal/registry"
)

type CustomColor struct {
	code string
}

func (c CustomColor) Print(a ...interface{})           { fmt.Print(c.code + fmt.Sprint(a...) + "\033[0m") }
func (c CustomColor) Printf(f string, a ...interface{}) { fmt.Printf(c.code+f+"\033[0m", a...) }
func (c CustomColor) Println(a ...interface{})         { fmt.Println(c.code + fmt.Sprint(a...) + "\033[0m") }
func (c CustomColor) Sprint(a ...interface{}) string   { return c.code + fmt.Sprint(a...) + "\033[0m" }

var (
	Purple = CustomColor{"\033[1;35m"}
	Pink   = CustomColor{"\033[35m"}
	Cyan   = CustomColor{"\033[36m"}
	Green  = CustomColor{"\033[32m"}
	Yellow = CustomColor{"\033[33m"}
	Red    = CustomColor{"\033[31m"}
	Blue   = CustomColor{"\033[34m"}
	HiWhite = "\033[97m"
	White  = "\033[37m"
)

func readKey() string {
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	defer exec.Command("stty", "-F", "/dev/tty", "echo").Run()

	var b = make([]byte, 3)
	n, _ := os.Stdin.Read(b)
	if n == 3 && b[0] == 27 && b[1] == 91 {
		switch b[2] {
		case 65: return "up"
		case 66: return "down"
		}
	}
	if b[0] == 9 { return "tab" }
	if b[0] == 13 || b[0] == 10 { return "enter" }
	if b[0] == 3 { return "ctrl+c" }
	if b[0] == 115 || b[0] == 83 { return "s" }
	if b[0] == 117 || b[0] == 85 { return "u" }
	return string(b[:n])
}

func readLine(defaultValue string) string {
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	defer exec.Command("stty", "-F", "/dev/tty", "echo").Run()

	var input []rune
	if defaultValue != "" {
		fmt.Print(defaultValue)
		input = []rune(defaultValue)
	}

	for {
		var b = make([]byte, 3)
		n, _ := os.Stdin.Read(b)
		if n == 1 {
			if b[0] == 13 || b[0] == 10 {
				fmt.Println()
				return string(input)
			}
			if b[0] == 127 || b[0] == 8 {
				if len(input) > 0 {
					input = input[:len(input)-1]
					fmt.Print("\b \b")
				}
				continue
			}
			if b[0] == 3 { os.Exit(0) }
			input = append(input, rune(b[0]))
			fmt.Print(string(b[0]))
		}
	}
}

func printBanner() {
	banner, err := os.ReadFile("internal/assets/banner.txt")
	if err == nil {
		Purple.Println(string(banner))
	}
	Cyan.Println(" > System established. Ready for infiltration...")
}

func customConfirm(label string) bool {
	if flagAuto || flagYes { return true }
	fmt.Printf("  %s [y/N]: ", Yellow.Sprint(label))
	input := strings.ToLower(readLine(""))
	return input == "y" || input == "yes"
}

func customInput(label, defaultValue string) string {
	if flagAuto || (label == "ARCHIVE_LOCATION" && flagYes) { return defaultValue }
	fmt.Printf("  %s: ", Cyan.Sprint(label))
	input := readLine(defaultValue)
	if input == "" { return defaultValue }
	return input
}

func clearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[1A\033[2K\r")
	}
}

func moveCursor(line, col int) {
	fmt.Printf("\033[%d;%dH", line, col)
}

func waitReturn() {
	if flagAuto || flagCLI { return }
	fmt.Print("\n\033[2m  (Press any key to return to mission control)\033[0m")
	readKey()
}

func truncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max { return s }
	return string(runes[:max-3]) + "..."
}

type AuditSummary struct {
	Logic, Integrity, Build, License string
}

func printAppReportDetailed(app registry.App, title string, buildExecuted bool, backupPath string, audit *AuditSummary) {
	width := 63
	boxColor := Blue
	if title == "AUDIT" { boxColor = Yellow } else if !buildExecuted { boxColor = Green }

	fmt.Println()
	drawBoxBorder(boxColor, fmt.Sprintf("%s // %s", title, app.Name), "┏", width)
	printBoxLine(boxColor, "IDENTITY", app.Name+" ("+app.Version+")", width)
	
	author := app.Author; if author == "" { parts := strings.Split(app.Repo, "/"); if len(parts) > 0 { author = parts[0] } }
	trustMarker := ""; if isAuthorTrusted(author) { trustMarker = Yellow.Sprint(" [★ TRUSTED]") }
	printBoxLine(boxColor, "AUTHOR", author+trustMarker, width)
	
	target := registry.ExpandPath(app.TargetPath, homeDir)
	if app.Type != "git-config" { target = filepath.Join(binDir, app.BinaryName) }
	printBoxLine(boxColor, "LOCATION", target, width)
	if backupPath != "" { printBoxLine(boxColor, "BACKUP_LOC", truncateString(backupPath, 45), width) }

	if audit != nil {
		fmt.Printf("  " + boxColor.Sprint("┠") + strings.Repeat(boxColor.Sprint("━"), width-4) + boxColor.Sprint("┨") + "\n")
		printBoxLine(boxColor, "LOGIC", audit.Logic, width); printBoxLine(boxColor, "INTEGRITY", audit.Integrity, width)
		trustStat := Green.Sprint("[✓] PINNED_COMMIT"); if app.CommitHash == "" { trustStat = Yellow.Sprint("[!] UNPINNED") }
		printBoxLine(boxColor, "TRUST", trustStat, width); printBoxLine(boxColor, "BUILD", audit.Build, width); printBoxLine(boxColor, "LICENSE", audit.License, width)
	}

	if buildExecuted && app.PostInstall != "" {
		fmt.Printf("  " + boxColor.Sprint("┠") + strings.Repeat(boxColor.Sprint("━"), width-4) + boxColor.Sprint("┨") + "\n")
		printBoxLine(boxColor, "STATUS", "EXECUTED_SUCCESSFULLY", width)
		printBoxWrapped(boxColor, "LOGIC", app.PostInstall, width)
	}
	drawBoxBorder(boxColor, "", "┗", width)
}

func visibleLen(s string) int {
	inCode := false; count := 0; runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == 27 { inCode = true; continue }; if inCode { if (runes[i] >= 'a' && runes[i] <= 'z') || (runes[i] >= 'A' && runes[i] <= 'Z') { inCode = false }; continue }; count++
	}
	return count
}

func drawBoxBorder(boxColor CustomColor, title, pos string, width int) {
	fmt.Print("  " + boxColor.Sprint(pos)); horizontalWidth := width - 4
	if title != "" {
		fmt.Print(boxColor.Sprint("━ ")); fmt.Print(HiWhite + title + " ")
		used := visibleLen(title) + 3; if used < horizontalWidth { fmt.Print(boxColor.Sprint(strings.Repeat("━", horizontalWidth-used))) }
	} else { fmt.Print(boxColor.Sprint(strings.Repeat("━", horizontalWidth))) }
	end := "┓"; switch pos { case "┗": end = "┛"; case "└": end = "┘"; case "┌": end = "┐" }
	fmt.Println(boxColor.Sprint(end))
}

func printBoxLine(boxColor CustomColor, label, value string, width int) {
	labelPart := "  " + label + ": "; prefixLen := visibleLen(labelPart); contentArea := width - 5 - prefixLen
	finalValue := truncateString(value, contentArea); vLen := visibleLen(finalValue); padding := ""
	if vLen < contentArea { padding = strings.Repeat(" ", contentArea-vLen) }
	fmt.Print("  " + boxColor.Sprint("┃") + labelPart + finalValue + padding + " " + boxColor.Sprint("┃") + "\n")
}

func printBoxWrapped(boxColor CustomColor, label, value string, width int) {
	labelPart := "  " + label + ": "; prefixLen := visibleLen(labelPart); contentArea := width - 5 - prefixLen
	cleanVal := strings.ReplaceAll(value, "\n", " ")
	if visibleLen(cleanVal) <= contentArea { printBoxLine(boxColor, label, cleanVal, width); return }
	var lines []string; words := strings.Fields(cleanVal); currentLine := ""
	for _, word := range words {
		testLine := currentLine; if testLine != "" { testLine += " " }; testLine += word
		if visibleLen(testLine) > contentArea {
			if currentLine != "" { lines = append(lines, currentLine); currentLine = word } else { lines = append(lines, truncateString(word, contentArea)); currentLine = "" }
		} else { currentLine = testLine }
	}
	if currentLine != "" { lines = append(lines, currentLine) }
	for i, line := range lines {
		lPart := labelPart; if i > 0 { lPart = strings.Repeat(" ", prefixLen) }
		vLen := visibleLen(line); padding := ""; if vLen < contentArea { padding = strings.Repeat(" ", contentArea-vLen) }
		fmt.Print("  " + boxColor.Sprint("┃") + lPart + line + padding + " " + boxColor.Sprint("┃") + "\n")
	}
}

func enterAltScreen() { fmt.Print("\033[?1049h\033[H") }
func exitAltScreen() { fmt.Print("\033[?1049l") }

func customSelect(apps []registry.App) {
	selected := 0; filter := ""; activeTab := 0; showNews := true; needsFullRedraw := true
	for {
		pendingUpdates := registry.GetPendingUpdates(loadSources(), baseDir, stateFile)
		var installedApps []registry.AppState; data, _ := os.ReadFile(stateFile); var allStates map[string]registry.AppState; json.Unmarshal(data, &allStates)
		for _, s := range allStates { installedApps = append(installedApps, s) }
		var filtered []registry.App
		if activeTab == 0 {
			query := strings.ToLower(filter)
			for _, a := range apps {
				author := a.Author; if author == "" { parts := strings.Split(a.Repo, "/"); if len(parts) > 0 { author = parts[0] } }
				if strings.Contains(strings.ToLower(a.Name), query) || strings.Contains(strings.ToLower(a.Description), query) || strings.Contains(strings.ToLower(author), query) { filtered = append(filtered, a) }
			}
		}
		if selected >= len(filtered) && activeTab == 0 { selected = 0 }
		if selected >= len(installedApps) && activeTab == 1 { selected = 0 }
		
		if needsFullRedraw {
			fmt.Print("\033[H\033[2J") // FULL CLEAR
			printBanner()
			Purple.Println(" ┌──────────────────────────────────────────────────────────┐")
			Purple.Print(" │ "); fmt.Printf("%-56s", "GOSIP OS // SYSTEM_V3.0"); Purple.Println(" │")
			status := "STATUS: ONLINE | REGISTRY_UNITS: " + fmt.Sprint(len(apps))
			Purple.Print(" │ "); fmt.Printf("%-56s", status); Purple.Println(" │")
			Purple.Println(" └──────────────────────────────────────────────────────────┘")
			needsFullRedraw = false
		}

		moveCursor(15, 1); fmt.Print("\033[J") // Safe start at line 15
		
		fmt.Println(); fmt.Print("  "); if activeTab == 0 { fmt.Printf("%s   %s\n", Blue.Sprint("▰▰ [ CORE_REGISTRY ] ▰▰"), HiWhite+"[ LOCAL_UNITS ]") } else { fmt.Printf("%s   %s\n", HiWhite+"[ CORE_REGISTRY ]", Pink.Sprint("▰▰ [ LOCAL_UNITS ] ▰▰")) }; fmt.Println()
		if activeTab == 0 { fmt.Printf("  %s %s%s\n", Cyan.Sprint("[SEARCH_REGISTRY]:"), HiWhite, filter+"_"); fmt.Println() }
		
		if showNews && len(pendingUpdates) > 0 {
			drawBoxBorder(Yellow, "SYSTEM_NEWS", "┏", 63); printBoxLine(Yellow, "ALERT", Yellow.Sprint(len(pendingUpdates))+" units have pending updates!", 63)
			for i, up := range pendingUpdates { if i >= 3 { printBoxLine(Yellow, "PLUS", fmt.Sprintf("... and %d more", len(pendingUpdates)-3), 63); break }; printBoxLine(Yellow, "UNIT", up.Name+" ("+up.Version+")", 63) }
			printBoxLine(Yellow, "ACTION", "Press 'U' to upgrade all units instantly.", 63); drawBoxBorder(Yellow, "", "┗", 63); fmt.Println()
		}

		if activeTab == 0 {
			if len(filtered) == 0 { Red.Println("    [!] NO_MATCHES_FOUND") }
			for i, app := range filtered {
				sIcon := White + "[○]"; sText := "AVAILABLE"; sColor := Blue; if state, exists := allStates[app.Name]; exists { if state.Version == app.Version { sIcon = Green.Sprint("[●]"); sText = "SYSTEM_READY"; sColor = Green } else { sIcon = Yellow.Sprint("[!]"); sText = "UPDATE_AVAIL"; sColor = Yellow } }
				author := app.Author; if author == "" { parts := strings.Split(app.Repo, "/"); if len(parts) > 0 { author = parts[0] } }; isTrusted := isAuthorTrusted(author); trustBadge := ""; if isTrusted { trustBadge = Yellow.Sprint(" [★]") }
				if i == selected {
					verStr := Pink.Sprint("(" + app.Version + ")"); for _, up := range pendingUpdates { if up.Name == app.Name { verStr += Yellow.Sprint(" [↑]"); break } }
					fmt.Printf("  " + Cyan.Sprint("➤ ") + "%s %s %s %s %s%s\n", Cyan.Sprint("["), HiWhite+app.Name+" "+verStr, Cyan.Sprint("]"), sIcon, sColor.Sprint(sText), trustBadge)
				} else {
					verStr := "(" + app.Version + ")"; for _, up := range pendingUpdates { if up.Name == app.Name { verStr += " [↑]"; break } }; dimTrust := ""; if isTrusted { dimTrust = " [★]" }
					fmt.Printf("    \033[2m%s %s %s %s%s\033[0m\n", app.Name, verStr, "[○]", sText, dimTrust)
				}
			}
		} else {
			if len(installedApps) == 0 { Yellow.Println("    [!] NO_UNITS_DEPLOYED_YET") }
			for i, app := range installedApps { if i == selected { fmt.Print("  " + Pink.Sprint("➤ ")); fmt.Printf("%s %s %s\n", Pink.Sprint("["), HiWhite+app.Name+" "+Green.Sprint("("+app.Version+")"), Pink.Sprint("]")) } else { fmt.Printf("    \033[2m%s (%s)\033[0m\n", app.Name, app.Version) } }
		}
		
		fmt.Println(); var currentApp *registry.App; if activeTab == 0 && len(filtered) > 0 { currentApp = &filtered[selected] }
		if currentApp != nil {
			boxColor := Cyan; if !currentApp.IsOfficial { boxColor = Yellow }; sIcon := White + "[○]"; sText := "AVAILABLE"; sColor := Blue; if state, exists := allStates[currentApp.Name]; exists { if state.Version == currentApp.Version { sIcon = Green.Sprint("[●]"); sText = "SYSTEM_READY"; sColor = Green } else { sIcon = Yellow.Sprint("[!]"); sText = "UPDATE_AVAIL"; sColor = Yellow } }
			drawBoxBorder(boxColor, "INFO", "┏", 63); printBoxWrapped(boxColor, "DESC", currentApp.Description, 63); author := currentApp.Author; if author == "" { parts := strings.Split(currentApp.Repo, "/"); if len(parts) > 0 { author = parts[0] } }; authorLine := author; isT := isAuthorTrusted(author); if isT { authorLine += Yellow.Sprint(" [★ TRUSTED_DEVELOPER]") }; printBoxLine(boxColor, "AUTH", authorLine, 63); printBoxLine(boxColor, "REPO", "github.com/"+currentApp.Repo, 63); printBoxLine(boxColor, "STAT", fmt.Sprintf("%-7s %s", sIcon, sColor.Sprint(sText)), 63); drawBoxBorder(boxColor, "", "┗", 63)
		} else if activeTab == 1 && len(installedApps) > 0 {
			app := installedApps[selected]; drawBoxBorder(Pink, "DEPLOYMENT_INFO", "┏", 63); printBoxLine(Pink, "NAME", app.Name+" ("+app.Version+")", 63); printBoxWrapped(Pink, "PATH", app.InstallPath, 63); printBoxLine(Pink, "DATE", app.InstallDate, 63); drawBoxBorder(Pink, "", "┗", 63)
		}
		fmt.Print("\n  ", Cyan.Sprint("[TAB]"), " Tab  ", Cyan.Sprint("[UP/DN]"), " Nav  ", Red.Sprint("[^C]"), " Abort"); if activeTab == 0 { fmt.Print("  ", Cyan.Sprint("[ENT]"), " Install  ", Yellow.Sprint("[S]"), " Shadow  ", Yellow.Sprint("[U]"), " Update All") } else { fmt.Print("  ", Red.Sprint("[U]"), " Uninstall") }
		
		key := readKey()
		if key == "tab" { activeTab = (activeTab + 1) % 2; selected = 0; showNews = true; needsFullRedraw = true; continue }
		if activeTab == 0 {
			if key == "u" { if len(pendingUpdates) > 0 { needsFullRedraw = true; for _, up := range pendingUpdates { installApp(up) }; showNews = true; continue } }
			if key == "s" { if len(filtered) > 0 { needsFullRedraw = true; enterShadowMode(filtered[selected]); continue } }
			if len(key) == 1 && key[0] >= 32 && key[0] <= 126 { filter += key; selected = 0; showNews = true } else if key == "backspace" || (len(key) == 1 && key[0] == 127) { if len(filter) > 0 { filter = filter[:len(filter)-1] }; showNews = true } else {
				switch key { 
				case "up": if selected > 0 { selected-- }
				case "down": if selected < len(filtered)-1 { selected++ }
				case "enter": if len(filtered) > 0 { needsFullRedraw = true; installApp(filtered[selected]) }
				case "ctrl+c": return 
				}
			}
		} else { 
			switch key { 
			case "up": if selected > 0 { selected-- }
			case "down": if selected < len(installedApps)-1 { selected++ }
			case "u": if len(installedApps) > 0 { needsFullRedraw = true; uninstallApp(installedApps[selected].Name) }
			case "ctrl+c": return 
			} 
		}
	}
}

func min(a, b int) int { if a < b { return a }; return b }
