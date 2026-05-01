package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// SyncRegistry handles the synchronization of official and community registries.
func SyncRegistry(defaultRegistry, communityRegistry, registryFile, communityFile, stateFile string, CurrentVersion string) {
	localSubmodulePath := "gosip-registry"

	if _, err := os.Stat(filepath.Join(localSubmodulePath, ".git")); err == nil {
		fmt.Println(" [SYS] SUBMODULE_DETECTED: Updating via Git...")
		
		cmd := exec.Command("git", "submodule", "update", "--remote", "--merge")
		err := cmd.Run()
		
		if err == nil {
			fmt.Println(" [✓] Submodule updated to latest.")
			
			copyRegistryFile(filepath.Join(localSubmodulePath, "registry.json"), registryFile)
			copyRegistryFile(filepath.Join(localSubmodulePath, "community.json"), communityFile)
			
			checkMainAppUpdate(registryFile, stateFile, CurrentVersion)
			return
		}
		fmt.Println(" [!] Git update failed. Falling back to HTTP sync...")
	}

	syncViaHTTP(defaultRegistry, communityRegistry, registryFile, communityFile, stateFile, CurrentVersion)
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
		if app.Name == "gosip" && app.Version != CurrentVersion {			// Note: selfUpdate is in main.go, not in registry package
			// selfUpdate(app.Version, app.Repo)
			return
		}
		if state, exists := allStates[app.Name]; exists && state.Version != app.Version {
			fmt.Printf(" [!] UPDATE_AVAILABLE: %s (%s -> %s)\n", app.Name, state.Version, app.Version)
		}
	}
}
