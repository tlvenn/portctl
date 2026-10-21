package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tlvenn/portctl/internal/client"
)

func TestLoadEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "org.env")
	content := "# org parameters\n\nDOMAIN=magelo.io\nexport ZFS_ROOT=rpool/data/magelo/ops\nQUOTED=\"a b\"\nSINGLE='c'\nEMPTY=\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := loadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"DOMAIN":   "magelo.io",
		"ZFS_ROOT": "rpool/data/magelo/ops",
		"QUOTED":   "a b",
		"SINGLE":   "c",
		"EMPTY":    "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLoadEnvFileRejectsLineWithoutEquals(t *testing.T) {
	path := filepath.Join(t.TempDir(), "org.env")
	if err := os.WriteFile(path, []byte("DOMAIN\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadEnvFile(path); err == nil {
		t.Error("expected an error for a line without '='")
	}
}

func TestMergeEnvKeepsUnmentionedKeys(t *testing.T) {
	env := []client.EnvVar{
		{Name: "JWT_SECRET", Value: "secret"},
		{Name: "DOMAIN", Value: "old.example"},
	}
	got := mergeEnv(env, map[string]string{"DOMAIN": "magelo.io", "ZFS_ROOT": "rpool/data/magelo/ops"})
	want := []client.EnvVar{
		{Name: "DOMAIN", Value: "magelo.io"},
		{Name: "JWT_SECRET", Value: "secret"},
		{Name: "ZFS_ROOT", Value: "rpool/data/magelo/ops"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
