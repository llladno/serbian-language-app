package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	for _, k := range []string{"DOTENV_TEST_A", "DOTENV_TEST_B", "DOTENV_TEST_C"} {
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range []string{"DOTENV_TEST_A", "DOTENV_TEST_B", "DOTENV_TEST_C"} {
			os.Unsetenv(k)
		}
	})

	path := filepath.Join(t.TempDir(), ".env")
	content := "" +
		"# a comment, and a blank line follow\n" +
		"\n" +
		"DOTENV_TEST_A=hello\n" +
		"DOTENV_TEST_B=\"quoted value\"\n" +
		"not a valid line without an equals sign\n" +
		"DOTENV_TEST_C=first\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	loadDotEnv(path)

	if got := os.Getenv("DOTENV_TEST_A"); got != "hello" {
		t.Errorf("DOTENV_TEST_A = %q, want %q", got, "hello")
	}
	if got := os.Getenv("DOTENV_TEST_B"); got != "quoted value" {
		t.Errorf("DOTENV_TEST_B = %q, want %q (surrounding quotes stripped)", got, "quoted value")
	}
	if got := os.Getenv("DOTENV_TEST_C"); got != "first" {
		t.Errorf("DOTENV_TEST_C = %q, want %q", got, "first")
	}
}

func TestLoadDotEnvNeverOverwritesRealEnv(t *testing.T) {
	os.Setenv("DOTENV_TEST_A", "real-value")
	t.Cleanup(func() { os.Unsetenv("DOTENV_TEST_A") })

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("DOTENV_TEST_A=from-dotenv\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	loadDotEnv(path)

	if got := os.Getenv("DOTENV_TEST_A"); got != "real-value" {
		t.Errorf("DOTENV_TEST_A = %q, want %q (a real env var must not be overwritten)", got, "real-value")
	}
}

func TestLoadDotEnvMissingFileIsNotAnError(t *testing.T) {
	loadDotEnv(filepath.Join(t.TempDir(), "does-not-exist.env"))
}
