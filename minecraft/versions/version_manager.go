package versions

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"
)

// MinecraftVersion represents a Minecraft version
type MinecraftVersion struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // release, snapshot, old_beta, old_alpha
	URL         string    `json:"url"`
	Time        time.Time `json:"time"`
	ReleaseTime time.Time `json:"releaseTime"`
	JavaVersion int       `json:"javaVersion"`
	Stable      bool      `json:"stable"`
}

// ServerType represents different server implementations
type ServerType struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description"`
	DownloadURL  string   `json:"downloadUrl"`
	Versions     []string `json:"versions"`
	Features     []string `json:"features"`
	JavaMin      int      `json:"javaMin"`
	JavaMax      int      `json:"javaMax"`
	Recommended  bool     `json:"recommended"`
	Performance  string   `json:"performance"` // Low, Medium, High, Beast
}

// VersionManager handles all Minecraft version operations
type VersionManager struct {
	CacheDir string
	client   *http.Client
}

// NewVersionManager creates a new version manager
func NewVersionManager(cacheDir string) *VersionManager {
	return &VersionManager{
		CacheDir: cacheDir,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAllVersions fetches all available Minecraft versions
func (vm *VersionManager) GetAllVersions() ([]MinecraftVersion, error) {
	resp, err := vm.client.Get("https://launchermeta.mojang.com/mc/game/version_manifest.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var manifest struct {
		Latest struct {
			Release  string `json:"release"`
			Snapshot string `json:"snapshot"`
		} `json:"latest"`
		Versions []MinecraftVersion `json:"versions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}

	// Add Java version compatibility
	for i := range manifest.Versions {
		manifest.Versions[i].JavaVersion = vm.getJavaVersion(manifest.Versions[i].ID)
		manifest.Versions[i].Stable = manifest.Versions[i].Type == "release"
	}

	return manifest.Versions, nil
}

// GetServerTypes returns all available server implementations
func (vm *VersionManager) GetServerTypes() []ServerType {
	return []ServerType{
		{
			Name:        "vanilla",
			DisplayName: "Vanilla",
			Description: "Official Minecraft server from Mojang",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4", "1.19.2", "1.18.2", "1.17.1", "1.16.5", "1.15.2", "1.14.4", "1.13.2", "1.12.2", "1.11.2", "1.10.2", "1.9.4", "1.8.9", "1.7.10"},
			Features:    []string{"Pure Minecraft", "No modifications", "Official support"},
			JavaMin:     8,
			JavaMax:     21,
			Recommended: false,
			Performance: "Medium",
		},
		{
			Name:        "paper",
			DisplayName: "Paper",
			Description: "High-performance Spigot fork with optimizations and bug fixes",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4", "1.19.2", "1.18.2", "1.17.1", "1.16.5", "1.15.2", "1.14.4", "1.13.2", "1.12.2", "1.11.2", "1.10.2", "1.9.4", "1.8.9"},
			Features:    []string{"Bukkit/Spigot plugins", "Performance optimizations", "Anti-cheat improvements", "Async chunk loading"},
			JavaMin:     17,
			JavaMax:     21,
			Recommended: true,
			Performance: "Beast",
		},
		{
			Name:        "purpur",
			DisplayName: "Purpur",
			Description: "Paper fork with additional features and configurability",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4", "1.19.2", "1.18.2", "1.17.1", "1.16.5", "1.15.2", "1.14.4", "1.13.2", "1.12.2"},
			Features:    []string{"Everything from Paper", "Extra configuration options", "Additional optimizations", "Custom features"},
			JavaMin:     17,
			JavaMax:     21,
			Recommended: true,
			Performance: "Beast",
		},
		{
			Name:        "fabric",
			DisplayName: "Fabric",
			Description: "Lightweight modding platform for Minecraft",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4", "1.19.2", "1.18.2", "1.17.1", "1.16.5", "1.15.2", "1.14.4"},
			Features:    []string{"Fabric mods", "Fast updates", "Lightweight", "Modern modding API"},
			JavaMin:     17,
			JavaMax:     21,
			Recommended: true,
			Performance: "High",
		},
		{
			Name:        "quilt",
			DisplayName: "Quilt",
			Description: "Fabric fork with additional features and improvements",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4", "1.19.2", "1.18.2", "1.17.1"},
			Features:    []string{"Fabric mod compatibility", "Enhanced mod loader", "Better performance", "Additional APIs"},
			JavaMin:     17,
			JavaMax:     21,
			Recommended: false,
			Performance: "High",
		},
		{
			Name:        "forge",
			DisplayName: "Minecraft Forge",
			Description: "The most popular modding platform for Minecraft",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4", "1.19.2", "1.18.2", "1.17.1", "1.16.5", "1.15.2", "1.14.4", "1.13.2", "1.12.2", "1.11.2", "1.10.2", "1.9.4", "1.8.9", "1.7.10"},
			Features:    []string{"Forge mods", "Extensive mod ecosystem", "Stable API", "Long-term support"},
			JavaMin:     8,
			JavaMax:     21,
			Recommended: true,
			Performance: "Medium",
		},
		{
			Name:        "neoforge",
			DisplayName: "NeoForge",
			Description: "Modern fork of Minecraft Forge with improvements",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1"},
			Features:    []string{"Forge mod compatibility", "Better performance", "Modern codebase", "Active development"},
			JavaMin:     17,
			JavaMax:     21,
			Recommended: true,
			Performance: "High",
		},
		{
			Name:        "spigot",
			DisplayName: "Spigot",
			Description: "Modified Minecraft server with plugin support",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4", "1.19.2", "1.18.2", "1.17.1", "1.16.5", "1.15.2", "1.14.4", "1.13.2", "1.12.2", "1.11.2", "1.10.2", "1.9.4", "1.8.9"},
			Features:    []string{"Bukkit plugins", "Performance improvements", "Configuration options", "Wide compatibility"},
			JavaMin:     8,
			JavaMax:     21,
			Recommended: false,
			Performance: "Medium",
		},
		{
			Name:        "bukkit",
			DisplayName: "CraftBukkit",
			Description: "Original plugin-supporting Minecraft server",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4", "1.19.2", "1.18.2", "1.17.1", "1.16.5", "1.15.2", "1.14.4", "1.13.2", "1.12.2"},
			Features:    []string{"Bukkit plugins", "Basic modifications", "Historical significance"},
			JavaMin:     8,
			JavaMax:     21,
			Recommended: false,
			Performance: "Low",
		},
		{
			Name:        "folia",
			DisplayName: "Folia",
			Description: "Paper fork designed for servers with high player counts",
			Versions:    []string{"1.21", "1.20.6", "1.20.4", "1.20.1", "1.19.4"},
			Features:    []string{"Regionised multithreading", "Ultra-high performance", "Massive player support", "Advanced optimization"},
			JavaMin:     17,
			JavaMax:     21,
			Recommended: true,
			Performance: "Beast",
		},
	}
}

// GetLatestVersion returns the latest stable version
func (vm *VersionManager) GetLatestVersion() (*MinecraftVersion, error) {
	versions, err := vm.GetAllVersions()
	if err != nil {
		return nil, err
	}

	for _, version := range versions {
		if version.Type == "release" {
			return &version, nil
		}
	}

	return nil, fmt.Errorf("no release version found")
}

// GetVersionsByType filters versions by type (release, snapshot, etc.)
func (vm *VersionManager) GetVersionsByType(versionType string) ([]MinecraftVersion, error) {
	versions, err := vm.GetAllVersions()
	if err != nil {
		return nil, err
	}

	var filtered []MinecraftVersion
	for _, version := range versions {
		if version.Type == versionType {
			filtered = append(filtered, version)
		}
	}

	return filtered, nil
}

// GetPopularVersions returns commonly used versions
func (vm *VersionManager) GetPopularVersions() []string {
	return []string{
		"1.21",     // Latest
		"1.20.6",   // Recent LTS
		"1.20.4",   // Stable
		"1.20.1",   // LTS
		"1.19.4",   // Popular
		"1.19.2",   // Modded favorite
		"1.18.2",   // Cave update
		"1.17.1",   // Stable
		"1.16.5",   // Nether update LTS
		"1.12.2",   // Modded classic
		"1.8.9",    // PvP favorite
		"1.7.10",   // Legacy modded
	}
}

// getJavaVersion returns the recommended Java version for a Minecraft version
func (vm *VersionManager) getJavaVersion(mcVersion string) int {
	version := strings.TrimPrefix(mcVersion, "1.")
	
	// Parse major version number
	if strings.Contains(version, ".") {
		parts := strings.Split(version, ".")
		if len(parts) > 0 {
			switch parts[0] {
			case "21", "20", "19", "18", "17":
				return 17 // Java 17+ for modern versions
			case "16", "15", "14", "13", "12":
				return 11 // Java 11+ for these versions
			default:
				return 8 // Java 8 for older versions
			}
		}
	}
	
	return 8 // Default to Java 8
}

// DownloadServerJar downloads the server jar for a specific version and type
func (vm *VersionManager) DownloadServerJar(serverType string, version string, outputPath string) error {
	downloadURL := vm.getDownloadURL(serverType, version)
	if downloadURL == "" {
		return fmt.Errorf("download URL not available for %s %s", serverType, version)
	}

	resp, err := vm.client.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(outputPath, data, 0644)
}

// getDownloadURL returns the download URL for a server type and version
func (vm *VersionManager) getDownloadURL(serverType string, version string) string {
	baseURLs := map[string]string{
		"paper":   "https://api.papermc.io/v2/projects/paper/versions/%s/builds/latest/downloads/paper-%s-latest.jar",
		"purpur":  "https://api.purpurmc.org/v2/purpur/%s/latest/download",
		"spigot":  "https://download.getbukkit.org/spigot/spigot-%s.jar",
		"bukkit":  "https://download.getbukkit.org/craftbukkit/craftbukkit-%s.jar",
		"fabric":  "https://meta.fabricmc.net/v2/versions/loader/%s/stable/server/jar",
		"forge":   "https://maven.minecraftforge.net/net/minecraftforge/forge/%s/forge-%s-installer.jar",
		"vanilla": "https://launcher.mojang.com/v1/objects/%s/server.jar",
	}

	if url, exists := baseURLs[serverType]; exists {
		return fmt.Sprintf(url, version, version)
	}

	return ""
}

// GetCompatiblePlugins returns plugins compatible with a version
func (vm *VersionManager) GetCompatiblePlugins(serverType string, version string) ([]string, error) {
	// This would integrate with the plugin marketplace
	compatibleTypes := map[string][]string{
		"paper":   {"bukkit", "spigot", "paper"},
		"purpur":  {"bukkit", "spigot", "paper", "purpur"},
		"spigot":  {"bukkit", "spigot"},
		"bukkit":  {"bukkit"},
		"fabric":  {"fabric"},
		"quilt":   {"fabric", "quilt"},
		"forge":   {"forge"},
		"neoforge": {"forge", "neoforge"},
		"folia":   {"bukkit", "spigot", "paper", "folia"},
	}

	return compatibleTypes[serverType], nil
}

// SortVersions sorts versions by release date (newest first)
func (vm *VersionManager) SortVersions(versions []MinecraftVersion) {
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].ReleaseTime.After(versions[j].ReleaseTime)
	})
}