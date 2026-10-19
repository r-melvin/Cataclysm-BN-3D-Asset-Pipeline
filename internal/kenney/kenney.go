package kenney

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//go:embed KenneyShapeProbe.cs KenneyShapeProbe.csproj
var adapterFiles embed.FS

type ExportOptions struct {
	GameDirectory string
	ProjectPath   string
	OutputPath    string
	Timeout       time.Duration
}

type installation struct {
	launchDir  string
	executable string
	melonDir   string
	modDir     string
	bootstrap  string
}

// Export loads a Kenney Shape project and writes a VOX model using the app's own voxel writer.
func Export(options ExportOptions) error {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		return fmt.Errorf("Kenney Shape export is not supported on %s", runtime.GOOS)
	}
	projectPath, err := filepath.Abs(options.ProjectPath)
	if err != nil {
		return fmt.Errorf("resolve project path: %w", err)
	}
	if strings.ToLower(filepath.Ext(projectPath)) != ".kenshape" {
		return fmt.Errorf("project path must use the .kenshape extension")
	}
	if info, err := os.Stat(projectPath); err != nil || info.IsDir() {
		return fmt.Errorf("project file does not exist: %s", projectPath)
	}
	outputPath, err := filepath.Abs(options.OutputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if !strings.EqualFold(filepath.Ext(outputPath), ".vox") {
		return fmt.Errorf("output path must use the .vox extension")
	}
	if options.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}
	gameRoot, err := filepath.Abs(options.GameDirectory)
	if err != nil {
		return fmt.Errorf("resolve game directory: %w", err)
	}
	game, err := locateInstallation(gameRoot)
	if err != nil {
		return err
	}
	if err := buildAdapter(game.melonDir, game.modDir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	logPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".kenney.log"
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()
	command := exec.CommandContext(ctx, game.executable, "-batchmode", "-nographics", "-logFile", logPath)
	command.Dir = game.launchDir
	command.Env = append(os.Environ(),
		"BN3D_KENNEY_PROJECT_PATH="+projectPath,
		"BN3D_KENNEY_EXPORT_PATH="+outputPath,
	)
	if runtime.GOOS == "linux" {
		command.Env = prependEnv(command.Env, "LD_LIBRARY_PATH", game.launchDir)
		command.Env = prependEnv(command.Env, "LD_PRELOAD", filepath.Base(game.bootstrap))
	} else if runtime.GOOS == "darwin" {
		command.Env = prependEnv(command.Env, "DYLD_LIBRARY_PATH", game.launchDir)
		command.Env = prependEnv(command.Env, "DYLD_INSERT_LIBRARIES", game.bootstrap)
	}
	output, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("Kenney Shape exceeded %s; inspect %s", options.Timeout, logPath)
	}
	if err != nil {
		return fmt.Errorf("Kenney Shape failed: %w; inspect %s\n%s", err, logPath, tail(string(output), 2000))
	}
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("Kenney Shape did not create the requested VOX file; inspect %s\n%s", logPath, tail(string(output), 2000))
	}
	return nil
}

func locateInstallation(gameRoot string) (installation, error) {
	return locateInstallationForOS(gameRoot, runtime.GOOS)
}

func locateInstallationForOS(gameRoot string, operatingSystem string) (installation, error) {
	if info, err := os.Stat(gameRoot); err != nil || !info.IsDir() {
		return installation{}, fmt.Errorf("game directory does not exist: %s", gameRoot)
	}
	launchDir := gameRoot
	gameBundle := ""
	if operatingSystem == "darwin" {
		if strings.EqualFold(filepath.Ext(gameRoot), ".app") {
			gameBundle = gameRoot
		} else {
			candidate := filepath.Join(gameRoot, "Kenney Shape.app")
			if _, err := os.Stat(candidate); err == nil {
				gameBundle = candidate
			}
		}
		if gameBundle != "" {
			launchDir = filepath.Join(gameBundle, "Contents", "MacOS")
		}
	}

	executableName := "Kenney Shape.x86_64"
	if operatingSystem == "windows" {
		executableName = "Kenney Shape.exe"
	} else if operatingSystem == "darwin" {
		executableName = "Kenney Shape"
	}
	executable := filepath.Join(launchDir, executableName)
	if !fileExists(executable) {
		return installation{}, fmt.Errorf("Kenney Shape executable not found in %s", launchDir)
	}
	dataCandidates := []string{
		filepath.Join(gameRoot, "Kenney Shape_Data", "Managed", "Assembly-CSharp.dll"),
		filepath.Join(launchDir, "Kenney Shape_Data", "Managed", "Assembly-CSharp.dll"),
	}
	if gameBundle != "" {
		dataCandidates = append(dataCandidates,
			filepath.Join(gameBundle, "Contents", "Resources", "Data", "Managed", "Assembly-CSharp.dll"),
			filepath.Join(gameBundle, "Contents", "Resources", "Kenney Shape_Data", "Managed", "Assembly-CSharp.dll"),
		)
	}
	managedAssembly := firstExisting(dataCandidates)
	if managedAssembly == "" {
		return installation{}, fmt.Errorf("Kenney Shape managed game assembly Assembly-CSharp.dll not found under %s", gameRoot)
	}
	melonRoot := firstDirectory([]string{filepath.Join(launchDir, "MelonLoader"), filepath.Join(gameRoot, "MelonLoader")})
	if melonRoot == "" {
		return installation{}, errors.New("MelonLoader is not installed in this Kenney Shape directory")
	}
	melonDLL, err := findMelonLoaderDLL(melonRoot)
	if err != nil {
		return installation{}, err
	}
	melonDir := filepath.Dir(melonDLL)
	modDir := filepath.Join(launchDir, "Mods")
	if operatingSystem == "linux" {
		bootstrap := firstExisting([]string{
			filepath.Join(launchDir, "MelonLoader.Bootstrap.so"),
			filepath.Join(gameRoot, "MelonLoader.Bootstrap.so"),
		})
		if bootstrap == "" {
			return installation{}, fmt.Errorf("MelonLoader.Bootstrap.so not found under %s", gameRoot)
		}
		return installation{launchDir: launchDir, executable: executable, melonDir: melonDir, modDir: modDir, bootstrap: bootstrap}, nil
	}
	if operatingSystem == "windows" {
		versionDLL := firstExisting([]string{filepath.Join(launchDir, "version.dll"), filepath.Join(gameRoot, "version.dll")})
		if versionDLL == "" {
			return installation{}, fmt.Errorf("MelonLoader version.dll proxy not found under %s; install MelonLoader in Kenney Shape first", gameRoot)
		}
		return installation{launchDir: launchDir, executable: executable, melonDir: melonDir, modDir: modDir}, nil
	}
	bootstrap := firstExisting([]string{
		filepath.Join(gameBundle, "Contents", "Frameworks", "MelonLoader.Bootstrap.dylib"),
		filepath.Join(gameRoot, "Contents", "Frameworks", "MelonLoader.Bootstrap.dylib"),
		filepath.Join(launchDir, "MelonLoader.Bootstrap.dylib"),
		filepath.Join(launchDir, "libMelonLoader.Bootstrap.dylib"),
		filepath.Join(gameRoot, "MelonLoader.Bootstrap.dylib"),
	})
	if bootstrap == "" {
		return installation{}, fmt.Errorf("MelonLoader macOS bootstrap dylib not found under %s; install a compatible MelonLoader build", gameRoot)
	}
	return installation{launchDir: launchDir, executable: executable, melonDir: melonDir, modDir: modDir, bootstrap: bootstrap}, nil
}

func findMelonLoaderDLL(root string) (string, error) {
	preferred := filepath.Join(root, "net35", "MelonLoader.dll")
	if fileExists(preferred) {
		return preferred, nil
	}
	var found string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == "MelonLoader.dll" && found == "" {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("search MelonLoader assemblies: %w", err)
	}
	if found == "" {
		return "", fmt.Errorf("MelonLoader.dll not found under %s", root)
	}
	return found, nil
}

func buildAdapter(melonDirectory string, modsDirectory string) error {
	workingDirectory, err := os.MkdirTemp("", "bn3d-kenney-adapter-")
	if err != nil {
		return fmt.Errorf("create adapter build directory: %w", err)
	}
	defer os.RemoveAll(workingDirectory)
	for _, name := range []string{"KenneyShapeProbe.cs", "KenneyShapeProbe.csproj"} {
		contents, err := adapterFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read embedded adapter file %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o644); err != nil {
			return fmt.Errorf("write adapter build file %s: %w", name, err)
		}
	}
	buildDirectory := filepath.Join(workingDirectory, "build")
	command := exec.Command("dotnet", "build", filepath.Join(workingDirectory, "KenneyShapeProbe.csproj"),
		"--configuration", "Release", "--output", buildDirectory,
		"-p:MELONLOADER_DIR="+melonDirectory, "--verbosity", "quiet")
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build Kenney adapter with dotnet: %w\n%s", err, tail(string(output), 4000))
	}
	if err := os.MkdirAll(modsDirectory, 0o755); err != nil {
		return fmt.Errorf("create MelonLoader Mods directory: %w", err)
	}
	mod, err := os.ReadFile(filepath.Join(buildDirectory, "KenneyShapeProbe.dll"))
	if err != nil {
		return fmt.Errorf("read built Kenney adapter: %w", err)
	}
	if err := os.WriteFile(filepath.Join(modsDirectory, "KenneyShapeProbe.dll"), mod, 0o644); err != nil {
		return fmt.Errorf("install Kenney adapter: %w", err)
	}
	return nil
}

func firstExisting(paths []string) string {
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}
	return ""
}

func firstDirectory(paths []string) string {
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func prependEnv(environment []string, key string, value string) []string {
	if value == "" {
		return environment
	}
	filtered := make([]string, 0, len(environment)+1)
	oldValue := ""
	prefix := key + "="
	for _, item := range environment {
		if strings.HasPrefix(item, prefix) {
			oldValue = strings.TrimPrefix(item, prefix)
		} else {
			filtered = append(filtered, item)
		}
	}
	if oldValue != "" {
		value += string(os.PathListSeparator) + oldValue
	}
	return append(filtered, prefix+value)
}

func tail(value string, length int) string {
	if len(value) <= length {
		return value
	}
	return value[len(value)-length:]
}
