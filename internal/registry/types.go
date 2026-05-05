package registry

/*
 * GOSIP REGISTRY - DATA MODELS
 * ----------------------------
 * File: internal/registry/types.go
 * Purpose: Defines the core data structures used throughout the system.
 */

type App struct {
	Name         string   `json:"name"`
	Author       string   `json:"author,omitempty"`
	AuthorNote   string   `json:"author_note,omitempty"`
	Type         string   `json:"type"`
	Description  string   `json:"description"`
	Repo         string   `json:"repo"`
	Version      string   `json:"version"`
	CommitHash   string   `json:"commit_hash,omitempty"`
	BinaryName   string   `json:"binary_name,omitempty"`
	DownloadURL  string   `json:"download_url,omitempty"`
	TargetPath   string   `json:"target_path,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	PostInstall  string   `json:"post_install,omitempty"`
	IsOfficial   bool     `json:"is_official"`
}

type Registry struct {
	Version string `json:"version"`
	Apps    []App  `json:"apps"`
}

type AppState struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	InstallPath string `json:"install_path"`
	BinPath     string `json:"bin_path"`
	InstallDate string `json:"install_date"`
}

type RegistrySource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	File string `json:"file"`
}
