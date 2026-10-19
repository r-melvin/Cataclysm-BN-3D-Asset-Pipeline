package kenney

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocateInstallationWindows(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "Kenney Shape.exe"))
	writeFixtureFile(t, filepath.Join(root, "Kenney Shape_Data", "Managed", "Assembly-CSharp.dll"))
	writeFixtureFile(t, filepath.Join(root, "MelonLoader", "net35", "MelonLoader.dll"))
	writeFixtureFile(t, filepath.Join(root, "version.dll"))

	game, err := locateInstallationForOS(root, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if game.executable != filepath.Join(root, "Kenney Shape.exe") || game.bootstrap != "" {
		t.Fatalf("unexpected Windows installation paths: %+v", game)
	}
}

func TestLocateInstallationMacBundle(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "Kenney Shape.app")
	launchDirectory := filepath.Join(bundle, "Contents", "MacOS")
	writeFixtureFile(t, filepath.Join(launchDirectory, "Kenney Shape"))
	writeFixtureFile(t, filepath.Join(bundle, "Contents", "Resources", "Data", "Managed", "Assembly-CSharp.dll"))
	writeFixtureFile(t, filepath.Join(launchDirectory, "MelonLoader", "net35", "MelonLoader.dll"))
	bootstrap := filepath.Join(bundle, "Contents", "Frameworks", "MelonLoader.Bootstrap.dylib")
	writeFixtureFile(t, bootstrap)

	game, err := locateInstallationForOS(root, "darwin")
	if err != nil {
		t.Fatal(err)
	}
	if game.executable != filepath.Join(launchDirectory, "Kenney Shape") || game.bootstrap != bootstrap {
		t.Fatalf("unexpected macOS installation paths: %+v", game)
	}
}

func TestLocateInstallationLinux(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "Kenney Shape.x86_64"))
	writeFixtureFile(t, filepath.Join(root, "Kenney Shape_Data", "Managed", "Assembly-CSharp.dll"))
	writeFixtureFile(t, filepath.Join(root, "MelonLoader", "net35", "MelonLoader.dll"))
	bootstrap := filepath.Join(root, "MelonLoader.Bootstrap.so")
	writeFixtureFile(t, bootstrap)

	game, err := locateInstallationForOS(root, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if game.executable != filepath.Join(root, "Kenney Shape.x86_64") || game.bootstrap != bootstrap {
		t.Fatalf("unexpected Linux installation paths: %+v", game)
	}
}

func writeFixtureFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
}
