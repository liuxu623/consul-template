package renderer

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	t.Run("parent_folder_missing", func(t *testing.T) {
		// Create a TempDir and a TempFile in that TempDir, then remove them to
		// "simulate" a non-existent folder
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(outDir); err != nil {
			t.Fatal(err)
		}

		if err := AtomicWrite(outFile.Name(), true, nil, 0644, -1); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(outFile.Name()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("retains_permissions", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		os.Chmod(outFile.Name(), 0600)

		if err := AtomicWrite(outFile.Name(), true, nil, 0, -1); err != nil {
			t.Fatal(err)
		}

		stat, err := os.Stat(outFile.Name())
		if err != nil {
			t.Fatal(err)
		}

		expected := os.FileMode(0600)
		if stat.Mode() != expected {
			t.Errorf("expected %q to be %q", stat.Mode(), expected)
		}
	})

	t.Run("non_existent", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(outDir)
		defer os.RemoveAll(outDir)

		// Try AtomicWrite to a file that doesn't exist yet
		file := filepath.Join(outDir, "nope/not/it/create")
		if err := AtomicWrite(file, true, nil, 0644, -1); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(file); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("non_existent_no_create", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(outDir)
		defer os.RemoveAll(outDir)

		// Try AtomicWrite to a file that doesn't exist yet
		file := filepath.Join(outDir, "nope/not/it/nope-no-create")
		if err := AtomicWrite(file, false, nil, 0644, -1); err != ErrNoParentDir {
			t.Fatalf("expected %q to be %q", err, ErrNoParentDir)
		}
	})

	t.Run("backup", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(outFile.Name(), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := outFile.Write([]byte("before")); err != nil {
			t.Fatal(err)
		}

		if err := AtomicWrite(outFile.Name(), true, []byte("after"), 0644, 1); err != nil {
			t.Fatal(err)
		}

		fileInfos, err := ioutil.ReadDir(outDir)
		if err != nil {
			t.Fatal(err)
		}
		var filename string
		base := filepath.Base(outFile.Name())
		for _, fileInfo := range fileInfos {
			if !fileInfo.IsDir() &&
				strings.HasPrefix(fileInfo.Name(), base+".") &&
				validUnixTimestamp.MatchString(fileInfo.Name()[len(base+"."):]) {
				filename = fileInfo.Name()
			}
		}
		f, err := ioutil.ReadFile(outDir + "/" + filename)
		if err != nil {
			t.Fatal(err.Error())
		}
		if !bytes.Equal(f, []byte("before")) {
			t.Fatalf("expected %q to be %q", f, []byte("before"))
		}

		if stat, err := os.Stat(outDir + "/" + filename); err != nil {
			t.Fatal(err)
		} else {
			if stat.Mode() != 0600 {
				t.Fatalf("expected %d to be %d", stat.Mode(), 0600)
			}
		}
	})

	t.Run("backup_not_exists", func(t *testing.T) {
		outDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(outDir)
		outFile, err := ioutil.TempFile(outDir, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(outFile.Name()); err != nil {
			t.Fatal(err)
		}

		if err := AtomicWrite(outFile.Name(), true, nil, 0644, 1); err != nil {
			t.Fatal(err)
		}

		// Shouldn't have a backup file, since the original file didn't exist
		fileInfos, err := ioutil.ReadDir(outDir)
		if err != nil {
			t.Fatal(err)
		}
		var filename string
		base := filepath.Base(outFile.Name())
		for _, fileInfo := range fileInfos {
			if !fileInfo.IsDir() &&
				strings.HasPrefix(fileInfo.Name(), base+".") &&
				validUnixTimestamp.MatchString(fileInfo.Name()[len(base+"."):]) {
				filename = fileInfo.Name()
				break
			}
		}
		if filename != "" {
			t.Fatalf("expected file %s not exists", filename)
		}
	})
}
