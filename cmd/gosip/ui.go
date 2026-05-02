package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

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
	fmt.Printf("  %s [y/N]: ", Yellow.Sprint(label))
	input := strings.ToLower(readLine(""))
	// Clear the prompt line (readLine already added a \n)
	fmt.Print("\033[A\033[2K\r")
	return input == "y" || input == "yes"
}

func customInput(label, defaultValue string) string {
	fmt.Printf("  %s [%s]: ", Cyan.Sprint(label), defaultValue)
	input := readLine(defaultValue)
	// Clear the prompt line
	fmt.Print("\033[A\033[2K\r")
	if input == "" { return defaultValue }
	return input
}

func clearLines(n int) {
	for i := 0; i < n; i++ {
		fmt.Print("\033[A\033[2K\r")
	}
}

func waitReturn() {
	fmt.Print(Cyan.Sprint("\n  Press [ENTER] to return to terminal..."))
	readKey()
}

func wrapText(text string, limit int) string {
	words := strings.Fields(text)
	if len(words) == 0 { return "" }
	var result strings.Builder
	lineLen := 0
	for i, word := range words {
		if lineLen+len(word) > limit {
			result.WriteString("\n        ")
			result.WriteString(word)
			lineLen = len(word)
		} else {
			if i > 0 { result.WriteString(" "); lineLen++ }
			result.WriteString(word)
			lineLen += len(word)
		}
	}
	return result.String()
}

func truncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

func printGridLine(label, value string, labelColor CustomColor, width int) {
	fmt.Print(Blue.Sprint(" │ "), labelColor.Sprint(label), " ")
	taken := 3 + len(label) + 1
	padding := width - taken - 1
	if padding < 0 { padding = 0 }
	fmt.Printf("%-*s", padding, value)
	Blue.Println("│")
}

func printGridLineDetailed(label, value string, labelColor CustomColor, width int) {
	fmt.Print("  ", Blue.Sprint("│ "), labelColor.Sprint(label), " ")
	labelLen := utf8.RuneCountInString(label)
	valueWidth := width - 2 - 2 - labelLen - 1 - 1
	if valueWidth < 0 { valueWidth = 0 }
	fmt.Printf("%-*s", valueWidth, truncateString(value, valueWidth))
	Blue.Println("│")
}

type AuditSummary struct {
	Logic     string
	Integrity string
	Build     string
	License   string
}

func printAppReportDetailed(app registry.App, title string, buildExecuted bool, backupPath string, audit *AuditSummary) {
	width := 63
	boxColor := Blue
	if title == "AUDIT" { boxColor = Yellow }
	if !buildExecuted && title != "AUDIT" { boxColor = Green }

	fmt.Println()
	// Header Section
	headerText := fmt.Sprintf(" %s // %s ", title, app.Name)
	boxColor.Print("  ┌─" + headerText)
	// Calculate exact line to fill the width
	headerLen := visibleLen(headerText)
	boxColor.Println(strings.Repeat("─", width-headerLen-5) + "┐")

	// Core Identity Block
	printBoxLine(boxColor, "IDENTITY", app.Name+" ("+app.Version+")", width)
	
	target := registry.ExpandPath(app.TargetPath, homeDir)
	if app.Type != "git-config" {
		target = filepath.Join(binDir, app.BinaryName)
	}
	printBoxLine(boxColor, "LOCATION", target, width)

	if backupPath != "" {
		printBoxLine(boxColor, "BACKUP_LOC", truncateString(backupPath, 45), width)
	}

	// Extended Info for Audits
	if title == "AUDIT" {
		printBoxLine(boxColor, "SOURCE", "github.com/"+app.Repo, width)
		deps := strings.Join(app.Dependencies, ", ")
		if deps == "" { deps = "None" }
		printBoxLine(boxColor, "REQS", deps, width)
	}

	// Deep Audit Summary Block
	if audit != nil {
		boxColor.Print("  ├─ AUDIT_REPORT ")
		boxColor.Println(strings.Repeat("─", width-19) + "┤")
		printBoxLine(boxColor, "LOGIC", audit.Logic, width)
		printBoxLine(boxColor, "INTEGRITY", audit.Integrity, width)
		printBoxLine(boxColor, "BUILD", audit.Build, width)
		printBoxLine(boxColor, "LICENSE", audit.License, width)
	}

	// Provisioning Info Block
	if buildExecuted && app.PostInstall != "" {
		boxColor.Print("  ├─ PROVISIONING ")
		boxColor.Println(strings.Repeat("─", width-19) + "┤")
		printBoxLine(boxColor, "STATUS", "EXECUTED_SUCCESSFULLY", width)
		printBoxLine(boxColor, "LOGIC", truncateString(app.PostInstall, 45), width)
	}

	boxColor.Println("  └" + strings.Repeat("─", width-4) + "┘")
}

func visibleLen(s string) int {
	// Strip ANSI escape codes to calculate visible length
	inCode := false
	count := 0
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == 27 { // ESC
			inCode = true
			continue
		}
		if inCode {
			if (runes[i] >= 'a' && runes[i] <= 'z') || (runes[i] >= 'A' && runes[i] <= 'Z') {
				inCode = false
			}
			continue
		}
		count++
	}
	return count
}

func drawBoxBorder(boxColor CustomColor, title, pos string, width int) {
	fmt.Print("  " + boxColor.Sprint(pos))
	
	// Total horizontal available = width - 2 (leading spaces) - 2 (corners) = width - 4
	horizontalWidth := width - 4
	
	if title != "" {
		fmt.Print(boxColor.Sprint("━ "))
		fmt.Print(HiWhite + title + " ")
		used := visibleLen(title) + 3 // "━ " (2) + title + " " (1)
		if used < horizontalWidth {
			fmt.Print(boxColor.Sprint(strings.Repeat("━", horizontalWidth-used)))
		}
	} else {
		fmt.Print(boxColor.Sprint(strings.Repeat("━", horizontalWidth)))
	}
	
	end := "┓"
	if pos == "┗" { end = "┛" }
	fmt.Println(boxColor.Sprint(end))
}

func printBoxLine(boxColor CustomColor, label, value string, width int) {
	labelPart := "  " + label + ": "
	prefixLen := visibleLen(labelPart)
	
	// Total available = width - 2 (leading spaces) - 1 (┃) - prefixLen - 1 (space before ┃) - 1 (┃)
	contentArea := width - 5 - prefixLen
	
	finalValue := truncateString(value, contentArea)
	vLen := visibleLen(finalValue)
	
	padding := ""
	if vLen < contentArea {
		padding = strings.Repeat(" ", contentArea-vLen)
	}
	
	fmt.Print("  " + boxColor.Sprint("┃") + labelPart + finalValue + padding + " " + boxColor.Sprint("┃") + "\n")
}

func printBoxWrapped(boxColor CustomColor, label, value string, width int) {
	labelPart := "  " + label + ": "
	prefixLen := visibleLen(labelPart)
	contentArea := width - 5 - prefixLen

	if visibleLen(value) <= contentArea {
		printBoxLine(boxColor, label, value, width)
		return
	}

	var lines []string
	words := strings.Fields(value)
	currentLine := ""
	
	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word
		
		if visibleLen(testLine) > contentArea {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Word itself is too long, force truncate it to avoid infinite loop
				lines = append(lines, truncateString(word, contentArea))
				currentLine = ""
			}
		} else {
			currentLine = testLine
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	for i, line := range lines {
		lPart := labelPart
		if i > 0 {
			lPart = strings.Repeat(" ", prefixLen)
		}
		
		vLen := visibleLen(line)
		padding := ""
		if vLen < contentArea {
			padding = strings.Repeat(" ", contentArea-vLen)
		}
		fmt.Print("  " + boxColor.Sprint("┃") + lPart + line + padding + " " + boxColor.Sprint("┃") + "\n")
	}
}

func customSelect(apps []registry.App) (int, bool) {
	selected := 0
	filter := ""
	activeTab := 0 // 0: CORE_REGISTRY, 1: LOCAL_UNITS
	
	for {
		// Load Installed Apps for Local Tab
		var installedApps []registry.AppState
		data, _ := os.ReadFile(stateFile)
		var allStates map[string]registry.AppState
		json.Unmarshal(data, &allStates)
		for _, s := range allStates {
			installedApps = append(installedApps, s)
		}

		// Filter for CORE tab
		var filtered []registry.App
		if activeTab == 0 {
			query := strings.ToLower(filter)
			isAuthorSearch := strings.HasPrefix(query, "a:") || strings.HasPrefix(query, "author:")
			isTypeSearch := strings.HasPrefix(query, "t:") || strings.HasPrefix(query, "type:")

			for _, a := range apps {
				match := false
				if isAuthorSearch {
					target := ""
					if strings.HasPrefix(query, "a:") { target = query[2:] } else { target = query[7:] }
					
					actualAuthor := a.Author
					if actualAuthor == "" {
						parts := strings.Split(a.Repo, "/")
						if len(parts) > 0 { actualAuthor = parts[0] }
					}
					if strings.Contains(strings.ToLower(actualAuthor), target) { match = true }
				} else if isTypeSearch {
					target := ""
					if strings.HasPrefix(query, "t:") { target = query[2:] } else { target = query[5:] }
					if strings.Contains(strings.ToLower(a.Type), target) { match = true }
				} else {
					if strings.Contains(strings.ToLower(a.Name), query) || strings.Contains(strings.ToLower(a.Description), query) {
						match = true
					}
				}
				
				if match { filtered = append(filtered, a) }
			}
		}

		if selected >= len(filtered) && activeTab == 0 { selected = 0 }
		if selected >= len(installedApps) && activeTab == 1 { selected = 0 }
		
		fmt.Print("\033[H\033[2J") // Clear
		printBanner()

		Purple.Println(" ┌──────────────────────────────────────────────────────────┐")
		Purple.Print(" │ ")
		fmt.Printf("%-56s", "GOSIP OS // SYSTEM_V3.0")
		Purple.Println(" │")
		Purple.Print(" │ ")
		if activeTab == 0 {
			fmt.Printf("%-56s", "STATUS: ONLINE | REGISTRY_UNITS: "+fmt.Sprint(len(apps)))
		} else {
			fmt.Printf("%-56s", "STATUS: LOCAL_STATION | DEPLOYED_UNITS: "+fmt.Sprint(len(installedApps)))
		}
		Purple.Println(" │")
		Purple.Println(" └──────────────────────────────────────────────────────────┘")
		fmt.Println()

		// Tab Header Area
		fmt.Print("  ")
		if activeTab == 0 {
			fmt.Printf("%s   %s\n", Blue.Sprint("▰▰ [ CORE_REGISTRY ] ▰▰"), HiWhite+"[ LOCAL_UNITS ]")
		} else {
			fmt.Printf("%s   %s\n", HiWhite+"[ CORE_REGISTRY ]", Pink.Sprint("▰▰ [ LOCAL_UNITS ] ▰▰"))
		}
		fmt.Println()

		if activeTab == 0 {
			fmt.Printf("  %s %s%s\n", Cyan.Sprint("[SEARCH_REGISTRY]:"), HiWhite, filter+"_")
			fmt.Println()
		}

		if activeTab == 0 {
			// Render Registry Tab
			if len(filtered) == 0 { Red.Println("    [!] NO_MATCHES_FOUND") }
			for i, app := range filtered {
				statusIcon := White + "[○]"
				statusText := "AVAILABLE"
				statusColor := Blue
				if state, exists := allStates[app.Name]; exists {
					if state.Version == app.Version {
						statusIcon = Green.Sprint("[●]")
						statusText = "SYSTEM_READY"
						statusColor = Green
					} else {
						statusIcon = Yellow.Sprint("[!]")
						statusText = "UPDATE_AVAIL"
						statusColor = Yellow
					}
				}

				if i == selected {
					fmt.Print("  " + Cyan.Sprint("➤ "))
					fmt.Printf("%s %s %s %s %s\n", 
						Cyan.Sprint("["), 
						HiWhite+app.Name+" "+Pink.Sprint("("+app.Version+")"), 
						Cyan.Sprint("]"),
						statusIcon,
						statusColor.Sprint(statusText))
				} else {
					fmt.Printf("    \033[2m%s (%s) %s %s\033[0m\n", app.Name, app.Version, "[○]", statusText)
				}
			}
		} else {
			// Render Local Tab
			if len(installedApps) == 0 { Yellow.Println("    [!] NO_UNITS_DEPLOYED_YET") }
			for i, app := range installedApps {
				if i == selected {
					fmt.Print("  " + Pink.Sprint("➤ "))
					fmt.Printf("%s %s %s\n", 
						Pink.Sprint("["), 
						HiWhite+app.Name+" "+Green.Sprint("("+app.Version+")"), 
						Pink.Sprint("]"))
				} else {
					fmt.Printf("    \033[2m%s (%s)\033[0m\n", app.Name, app.Version)
				}
			}
		}

		// Information Panel at the bottom
		fmt.Println()
		var currentApp *registry.App
		if activeTab == 0 && len(filtered) > 0 {
			currentApp = &filtered[selected]
		}
		
		boxWidth := 63
		if currentApp != nil {
			boxColor := Cyan
			if !currentApp.IsOfficial { boxColor = Yellow }
			
			statusIcon := White + "[○]"
			statusText := "AVAILABLE"
			statusColor := Blue
			if state, exists := allStates[currentApp.Name]; exists {
				if state.Version == currentApp.Version {
					statusIcon = Green.Sprint("[●]")
					statusText = "SYSTEM_READY"
					statusColor = Green
				} else {
					statusIcon = Yellow.Sprint("[!]")
					statusText = "UPDATE_AVAIL"
					statusColor = Yellow
				}
			}

			drawBoxBorder(boxColor, "INFO", "┏", boxWidth)
			printBoxWrapped(boxColor, "DESC", currentApp.Description, boxWidth)
			printBoxLine(boxColor, "REPO", "github.com/"+currentApp.Repo, boxWidth)
			
			statusBase := fmt.Sprintf("%-7s %s", statusIcon, statusColor.Sprint(statusText))
			printBoxLine(boxColor, "STAT", statusBase, boxWidth)
			
			drawBoxBorder(boxColor, "", "┗", boxWidth)
		} else if activeTab == 1 && len(installedApps) > 0 {
			app := installedApps[selected]
			drawBoxBorder(Pink, "DEPLOYMENT_INFO", "┏", boxWidth)
			
			printBoxLine(Pink, "NAME", app.Name+" ("+app.Version+")", boxWidth)
			printBoxWrapped(Pink, "PATH", app.InstallPath, boxWidth)
			printBoxLine(Pink, "DATE", app.InstallDate, boxWidth)
			
			drawBoxBorder(Pink, "", "┗", boxWidth)
		}

		// Bottom Controls Info
		fmt.Print("\n  ", Cyan.Sprint("[TAB]"), " Switch Tab  ", Cyan.Sprint("[UP/DOWN]"), " Nav  ", Red.Sprint("[CTRL+C]"), " Abort")
		if activeTab == 0 {
			fmt.Print("  ", Cyan.Sprint("[ENTER]"), " Install  ", Yellow.Sprint("[S]"), " Shadow")
		} else {
			fmt.Print("  ", Red.Sprint("[U]"), " Uninstall")
		}

		key := readKey()
		if key == "tab" || (len(key) == 1 && key[0] == 9) {
			activeTab = (activeTab + 1) % 2
			selected = 0
			continue
		}

		if activeTab == 0 {
			// Registry Keys
			if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
				if key == "s" || key == "S" {
					if len(filtered) > 0 {
						for i, a := range apps {
							if a.Name == filtered[selected].Name { return i, true }
						}
					}
				}
				filter += key
				selected = 0
			} else if key == "backspace" || (len(key) == 1 && key[0] == 127) {
				if len(filter) > 0 { filter = filter[:len(filter)-1] }
			} else {
				switch key {
				case "up": if selected > 0 { selected-- }
				case "down": if selected < len(filtered)-1 { selected++ }
				case "enter":
					if len(filtered) > 0 {
						for i, a := range apps {
							if a.Name == filtered[selected].Name { return i, false }
						}
					}
				case "ctrl+c": fmt.Print("\033[H\033[2J"); os.Exit(0)
				}
			}
		} else {
			// Local Keys
			switch key {
			case "up": if selected > 0 { selected-- }
			case "down": if selected < len(installedApps)-1 { selected++ }
			case "u", "U":
				if len(installedApps) > 0 {
					uninstallApp(installedApps[selected].Name)
					waitReturn()
				}
			case "ctrl+c": fmt.Print("\033[H\033[2J"); os.Exit(0)
			}
		}
	}
}
