package storage

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const (
	passphrase    = "Staatsgeheimnis"
	newPassphrase = "Nur für den Dienstgebrauch"
)

var (
	data   = []byte("cleartext")
	update = []byte("updated cleartext")
)

func TestCreateOpenEmpty(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "storage_test")
	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "storage_test")
	_, err = Create(filename, passphrase, nil)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	_, _, err = Open(filename, passphrase)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
}

func TestCreateOpenData(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "storage_test")
	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "storage_test")
	_, err = Create(filename, passphrase, data)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	_, out, err := Open(filename, passphrase)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Error("out != data")
	}
}

func TestCreateSaveOpen(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "storage_test")
	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "storage_test")
	s, err := Create(filename, passphrase, data)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	err = s.Save(update)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	_, out, err := Open(filename, passphrase)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	if !bytes.Equal(out, update) {
		t.Error("out != update")
	}
}

func TestCreateRekeyOpen(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "storage_test")
	if err != nil {
		t.Fatalf("TempDir() failed: %v", err)
	}
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, "storage_test")
	_, err = Create(filename, passphrase, data)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	err = Rekey(filename, passphrase, newPassphrase)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	_, _, err = Open(filename, passphrase)
	if err == nil {
		t.Error("Open() should fail")
	}
	_, out, err := Open(filename, newPassphrase)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Error("out != data")
	}
}
