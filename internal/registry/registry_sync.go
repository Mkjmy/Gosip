package registry

/*
 * GOSIP REGISTRY - SYNCHRONIZATION
 * --------------------------------
 * File: internal/registry/registry_sync.go
 * Purpose: Handles fetching and updating registry data from remote sources.
 *
 * Sections:
 * - [16-65]: Synchronization Logic (HTTP fetching with Cache-busting)
 * - [67-85]: File-system persistence for registry JSONs
 * - [87-120]: Version Checking (Pending updates for apps)
 * - [122-150]: Core Application Self-Update Checking
 */

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SyncRegistry handles the synchronization of all configured registries.
func SyncRegistry(sources []RegistrySource, baseDir, stateFile, CurrentVersion string) {
	for _, src := range sources {
		destFile := filepath.Join(baseDir, src.File)
		fmt.Printf(" [SYS] SYNCING_%s... ", strings.ToUpper(src.Name))
		
		cacheBuster := fmt.Sprintf("?t=%d", time.Now().Unix())
		if downloadToConfig(src.URL+cacheBuster, destFile) {
			fmt.Println("SUCCESS")
		} else {
			fmt.Println("FAILED")
		}
	}
	
	// Check for main app update from the 'official' registry if exists
	for _, src := range sources {
		if src.Name == "official" {
			checkMainAppUpdate(filepath.Join(baseDir, src.File), stateFile, CurrentVersion)
			break
		}
	}
}

func syncViaHTTP(defaultRegistry, communityRegistry, registryFile, communityFile, stateFile string, CurrentVersion string) {
	cacheBuster := fmt.Sprintf("?t=%d", time.Now().Unix())

	fmt.Print(" [SYS] SYNCING_OFFICIAL... ")
	if downloadToConfig(defaultRegistry+cacheBuster, registryFile) {
		fmt.Println("SUCCESS")
	} else {
		fmt.Println("FAILED")
	}

	fmt.Print(" [SYS] SYNCING_COMMUNITY... ")
	if downloadToConfig(communityRegistry+cacheBuster, communityFile) {
		fmt.Println("SUCCESS")
	} else {
		fmt.Println("FAILED")
	}

	checkMainAppUpdate(registryFile, stateFile, CurrentVersion)
}

func downloadToConfig(url, dest string) bool {
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return false
	}
	defer resp.Body.Close()

	// Ensure destination directory exists
	os.MkdirAll(filepath.Dir(dest), 0755)

	out, err := os.Create(dest)
	if err != nil {
		return false
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err == nil
}

func copyRegistryFile(src, dst string) {
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Printf(" [!] ERROR_READING: %s\n", src)
		return
	}
	// Ensure destination directory exists
	os.MkdirAll(filepath.Dir(dst), 0755)
	err = os.WriteFile(dst, data, 0644)
	if err != nil {
		fmt.Printf(" [!] ERROR_WRITING: %s\n", dst)
		return
	}
	fmt.Printf("  -> Synchronized: %s\n", filepath.Base(src))
}

func GetPendingUpdates(sources []RegistrySource, baseDir, stateFile string) []App {
	var updates []App
	
	allStates := make(map[string]AppState)
	stateData, err := os.ReadFile(stateFile)
	if err == nil {
		json.Unmarshal(stateData, &allStates)
	}
	if len(allStates) == 0 {
		return updates
	}

	for _, src := range sources {
		data, err := os.ReadFile(filepath.Join(baseDir, src.File))
		if err == nil {
			var reg Registry
			json.Unmarshal(data, &reg)
			for _, app := range reg.Apps {
				if state, exists := allStates[app.Name]; exists {
					if state.Version != app.Version {
						updates = append(updates, app)
					}
				}
			}
		}
	}
	
	return updates
}

func checkMainAppUpdate(registryFile, stateFile, CurrentVersion string) {
	data, err := os.ReadFile(registryFile)
	if err != nil {
		return
	}

	var reg Registry
	json.Unmarshal(data, &reg)

	allStates := make(map[string]AppState)
	stateData, _ := os.ReadFile(stateFile)
	json.Unmarshal(stateData, &allStates)

	for _, app := range reg.Apps {
		if app.Name == "gosip" && app.Version != CurrentVersion {
			return
		}
		if state, exists := allStates[app.Name]; exists && state.Version != app.Version {
			fmt.Printf(" [!] UPDATE_AVAILABLE: %s (%s -> %s)\n", app.Name, state.Version, app.Version)
		}
	}
}
