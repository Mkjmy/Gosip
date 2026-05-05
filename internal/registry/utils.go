package registry

/*
 * GOSIP REGISTRY - UTILITIES
 * --------------------------
 * File: internal/registry/utils.go
 * Purpose: Helper functions for file management, state persistence, and progress visualization.
 *
 * Sections:
 * - [15-30]: Path Manipulation & Backups
 * - [32-68]: State Management (Save/Get/Remove installed apps state)
 * - [70-150]: Progress Visualization (Dynamic Progress Bars, Download Counters)
 * - [152-165]: Network Helpers (GitHub repo validation)
 */

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Re-defining colors or accepting them as dependencies might be needed,
// but for now, let's keep it simple by removing direct color prints if possible or move color defs.
// Actually, let's just make it compilable.
func ExpandPath(path string, homeDir string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func MoveToBackup(src, dest string) error {
	if _, err := os.Stat(src); err == nil {
		os.MkdirAll(filepath.Dir(dest), 0755)
		// Try rename first (instant)
		if err := os.Rename(src, dest); err != nil {
			// Fallback to copy + remove for cross-partition moves
			cmd := exec.Command("cp", "-r", src, dest)
			if err := cmd.Run(); err != nil {
				return err
			}
			return os.RemoveAll(src)
		}
		return nil
	}
	return nil
}

func SaveState(state AppState, stateFile string) {
	allStates := make(map[string]AppState)
	data, err := os.ReadFile(stateFile)
	if err == nil {
		json.Unmarshal(data, &allStates)
	}
	state.InstallDate = time.Now().Format("2006-01-02 15:04:05")
	allStates[state.Name] = state
	newData, _ := json.MarshalIndent(allStates, "", "  ")
	os.WriteFile(stateFile, newData, 0644)
}

func GetState(appName string, stateFile string) (AppState, bool) {
	allStates := make(map[string]AppState)
	data, err := os.ReadFile(stateFile)
	if err == nil {
		json.Unmarshal(data, &allStates)
		state, exists := allStates[appName]
		return state, exists
	}
	return AppState{}, false
}

func RemoveState(appName string, stateFile string) {
	allStates := make(map[string]AppState)
	data, err := os.ReadFile(stateFile)
	if err == nil {
		json.Unmarshal(data, &allStates)
		delete(allStates, appName)
		newData, _ := json.MarshalIndent(allStates, "", "  ")
		os.WriteFile(stateFile, newData, 0644)
	}
}

type WriteCounter struct {
	Total   int64
	Current int64
	Label   string
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Current += int64(n)
	wc.printProgress()
	return n, nil
}

func (wc *WriteCounter) printProgress() {
	width := 30
	percent := float64(wc.Current) / float64(wc.Total)
	filled := int(percent * float64(width))
	if filled > width { filled = width }
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	// Consistently use 2 spaces and 20 chars for labels
	fmt.Printf("\r  %-20s [%s] %.0f%%", wc.Label, bar, percent*100)
	if wc.Current == wc.Total { fmt.Println() }
}

func ShowDynamicProgress(label string) (chan bool, chan bool) {
	done := make(chan bool)
	wait := make(chan bool)
	width := 30
	
	go func() {
		percent := 0
		baseDelay := 60 * time.Millisecond
		
		finish := func(curr int) {
			for p := curr; p <= 100; p++ {
				bar := strings.Repeat("█", (p*width)/100) + strings.Repeat("░", width-(p*width)/100)
				fmt.Printf("\r  %-20s [%s] %d%%", label, bar, p)
				time.Sleep(5 * time.Millisecond)
			}
			fmt.Println() // End the line properly
			wait <- true
		}

		for {
			select {
			case <-done:
				finish(percent)
				return
			default:
				if percent < 99 {
					percent++
					
					delay := baseDelay
					if percent > 50 { delay = time.Duration(float64(baseDelay) * 1.5) }
					if percent > 75 { delay = time.Duration(float64(baseDelay) * 3) }
					if percent > 85 { delay = time.Duration(float64(baseDelay) * 6) }
					if percent > 92 { delay = time.Duration(float64(baseDelay) * 15) }
					if percent > 97 { delay = time.Duration(float64(baseDelay) * 40) }

					bar := strings.Repeat("█", (percent*width)/100) + strings.Repeat("░", width-(percent*width)/100)
					fmt.Printf("\r  %-20s [%s] %d%%", label, bar, percent)
					
					innerTicker := time.NewTicker(delay)
					select {
					case <-done:
						innerTicker.Stop()
						finish(percent)
						return
					case <-innerTicker.C:
						innerTicker.Stop()
					}
				} else {
					// Stall at 99% and wait for the actual finish signal
					<-done
					finish(99)
					return
				}
			}
		}
	}()
	
	return done, wait
}

func CheckRepoExists(repo string) bool {
	url := fmt.Sprintf("https://github.com/%s", repo)
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
