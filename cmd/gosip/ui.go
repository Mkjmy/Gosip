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
	if len(s) <= max { return s }
	return s[:max-3] + "..."
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

func printAppReportDetailed(app registry.App, title string, buildExecuted bool, backupPath string) {
	width := 61
	boxColor := Blue
	if title == "AUDIT" { boxColor = Yellow }
	if !buildExecuted && title != "AUDIT" { boxColor = Green }

	fmt.Println()
	// Header Section
	headerText := fmt.Sprintf(" %s // %s ", title, app.Name)
	boxColor.Print("  ┌─" + headerText)
	// Calculate exact line to fill the width
	headerLen := utf8.RuneCountInString(headerText)
	boxColor.Println(strings.Repeat("─", width-headerLen-5) + "┐")

	// Core Identity Block
	printGridLineDetailed("IDENTITY:", app.Name+" ("+app.Version+")", Pink, width)
	
	target := registry.ExpandPath(app.TargetPath, homeDir)
	if app.Type != "git-config" {
		target = filepath.Join(binDir, app.BinaryName)
	}
	printGridLineDetailed("LOCATION:", target, Pink, width)

	if backupPath != "" {
		printGridLineDetailed("BACKUP_LOC:", truncateString(backupPath, 45), Pink, width)
	}

	// Extended Info for Audits
	if title == "AUDIT" {
		printGridLineDetailed("SOURCE:  ", "github.com/"+app.Repo, Pink, width)
		deps := strings.Join(app.Dependencies, ", ")
		if deps == "" { deps = "None" }
		printGridLineDetailed("REQS:    ", deps, Pink, width)
	}

	// Provisioning Info Block
	if buildExecuted && app.PostInstall != "" {
		boxColor.Print("  ├─ PROVISIONING ")
		boxColor.Println(strings.Repeat("─", width-19) + "┤")
		printGridLineDetailed("STATUS:  ", "EXECUTED_SUCCESSFULLY", Pink, width)
		printGridLineDetailed("LOGIC:   ", truncateString(app.PostInstall, 45), Pink, width)
	}

	boxColor.Println("  └" + strings.Repeat("─", width-4) + "┘")
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
		
		// Tab Header Area
		fmt.Print("  ")
		if activeTab == 0 {
			fmt.Printf("%s   %s\n", Blue.Sprint("▰▰ [ CORE_REGISTRY ] ▰▰"), HiWhite+"[ LOCAL_UNITS ]")
		} else {
			fmt.Printf("%s   %s\n", HiWhite+"[ CORE_REGISTRY ]", Pink.Sprint("▰▰ [ LOCAL_UNITS ] ▰▰"))
		}
		
		Purple.Println(" ┌──────────────────────────────────────────────────────────┐")
		Purple.Print(" │ ")
		fmt.Printf("%-56s", "GOSIP OS // CORE_SYSTEM_V3.0")
		Purple.Println(" │")
		Purple.Println(" └──────────────────────────────────────────────────────────┘")
		
		if activeTab == 0 {
			fmt.Printf("  %s %s%s\n", Cyan.Sprint("[SEARCH_REGISTRY]:"), HiWhite, filter+"_")
		} else {
			Pink.Printf("  [INSTALLED_UNITS]: %d units found\n", len(installedApps))
		}

		fmt.Println()
		if activeTab == 0 {
			// Render Registry Tab
			if len(filtered) == 0 { Red.Println("    [!] NO_MATCHES_FOUND") }
			for i, app := range filtered {
				statusNote := ""
				if state, exists := allStates[app.Name]; exists {
					if state.Version == app.Version {
						statusNote = Green.Sprint(" [INSTALLED]")
					} else {
						statusNote = Yellow.Sprint(" [UPDATE_AVAIL]")
					}
				}

				if i == selected {
					fmt.Print("  ")
					if app.IsOfficial { Cyan.Print("▶ ") } else { Yellow.Print("▶ ") }
					Purple.Print("[ ")
					fmt.Printf("%s%-20s", HiWhite, app.Name)
					Purple.Print(" ]")
					fmt.Print(statusNote, "  ")
					if app.IsOfficial { Blue.Println("← SYSTEM_READY") } else { Yellow.Println("← [!] UNVERIFIED") }
				} else {
					nameWidth := utf8.RuneCountInString(app.Name)
					fmt.Printf("    %s", app.Name)
					if nameWidth < 20 { fmt.Print(strings.Repeat(" ", 20-nameWidth)) }
					fmt.Print(statusNote, "    \n")
				}
			}
		} else {
			// Render Local Tab
			if len(installedApps) == 0 { Yellow.Println("    [!] NO_UNITS_DEPLOYED_YET") }
			for i, app := range installedApps {
				if i == selected {
					fmt.Printf("  %s %s (%s)\n", Pink.Sprint("▶"), HiWhite+app.Name, Green.Sprint(app.Version))
					fmt.Printf("    %s %s\n", Blue.Sprint("└─"), Yellow.Sprint(truncateString(app.InstallPath, 45)))
				} else {
					fmt.Printf("    %s %s\n", app.Name, Blue.Sprint("("+app.Version+")"))
				}
			}
		}

		// Information Panel for Registry
		if activeTab == 0 && len(filtered) > 0 {
			app := filtered[selected]
			fmt.Println()
			Blue.Println(" ┌─ AUDIT_LOG ──────────────────────────────────────────────┐")
			printGridLine("SOURCE:  ", "github.com/"+app.Repo, Pink, 61)
			printGridLine("VERSION: ", app.Version, Pink, 61)
			Blue.Println(" └──────────────────────────────────────────────────────────┘")
		}
		
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
