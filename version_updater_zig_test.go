package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleZigZon = `{
    .name = .mpb,
    .version = "0.1.5",
    .fingerprint = 0x33cdf8e3197da166,
    .minimum_zig_version = "0.17.0",
    .paths = .{
        "build.zig",
        "build.zig.zon",
    },
}
`

func TestUpdateZigVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "build.zig.zon")
	if err := os.WriteFile(path, []byte(sampleZigZon), 0644); err != nil {
		t.Fatalf("write build.zig.zon: %v", err)
	}

	modified, err := updateZigVersion(path, "1.2.3")
	if err != nil {
		t.Fatalf("updateZigVersion: %v", err)
	}
	if !modified {
		t.Fatal("expected build.zig.zon to be modified")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read build.zig.zon: %v", err)
	}
	text := string(got)
	if !strings.Contains(text, `.version = "1.2.3",`) {
		t.Errorf("top-level version not updated:\n%s", text)
	}
	if strings.Contains(text, `"0.1.5"`) {
		t.Errorf("old version still present:\n%s", text)
	}
	// 其他字段与版本号无关，必须原样保留。
	if !strings.Contains(text, `.fingerprint = 0x33cdf8e3197da166`) {
		t.Errorf("fingerprint changed:\n%s", text)
	}
	if !strings.Contains(text, `.minimum_zig_version = "0.17.0"`) {
		t.Errorf("minimum_zig_version changed:\n%s", text)
	}
}

// 只替换第一个顶层 .version：.dependencies 里的同名字段不能动，CRLF 不能变。
func TestUpdateZigVersionKeepsNestedAndCRLF(t *testing.T) {
	lines := []string{
		".{",
		`    .version = "0.1.5",`,
		"    .dependencies = .{",
		"        .demo = .{",
		`            .version = "9.9.9",`,
		"        },",
		"    },",
		"}",
		"",
	}
	path := filepath.Join(t.TempDir(), "build.zig.zon")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\r\n")), 0644); err != nil {
		t.Fatalf("write build.zig.zon: %v", err)
	}

	if _, err := updateZigVersion(path, "2.0.0"); err != nil {
		t.Fatalf("updateZigVersion: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read build.zig.zon: %v", err)
	}
	if !bytes.Contains(got, []byte(`.version = "2.0.0",`)) {
		t.Errorf("top-level version not updated:\n%s", got)
	}
	if !bytes.Contains(got, []byte(`.version = "9.9.9",`)) {
		t.Errorf("nested dependency version must not change:\n%s", got)
	}
	if bytes.Contains(bytes.ReplaceAll(got, []byte("\r\n"), nil), []byte("\n")) {
		t.Errorf("mixed line endings after update:\n%q", got)
	}
}

const noVersionZon = `{
    .name = .mpb,
}
`

func TestUpdateZigVersionMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "build.zig.zon")
	if err := os.WriteFile(path, []byte(noVersionZon), 0644); err != nil {
		t.Fatalf("write build.zig.zon: %v", err)
	}

	modified, err := updateZigVersion(path, "1.2.3")
	if err != nil {
		t.Fatalf("updateZigVersion: %v", err)
	}
	if modified {
		t.Fatal("file without .version must not be modified")
	}
}

func TestZigUpdaterSupported(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	updater := NewZigUpdater(dir)
	if updater.Supported() {
		t.Fatal("Supported() must be false without build.zig.zon")
	}

	if err := os.WriteFile("build.zig.zon", []byte(sampleZigZon), 0644); err != nil {
		t.Fatalf("write build.zig.zon: %v", err)
	}
	if !updater.Supported() {
		t.Fatal("Supported() must be true with build.zig.zon")
	}

	files, err := updater.Update("3.0.0")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(files) != 1 || files[0] != zigZonFilename {
		t.Fatalf("unexpected files: %v", files)
	}
}
