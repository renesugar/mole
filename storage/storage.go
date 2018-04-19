// Package storage provides encrypted file storage.
package storage

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

// Version of storage format.
const Version = 0x01

// State of storage (for later save operations).
type State struct {
	filename string   // original filename
	salt     [32]byte // salt for KDF
	nonce    [24]byte // nonce for secretbox
	key      [32]byte // derived key
}

func (s *State) save(w io.Writer, data []byte) error {
	// encrypt data
	enc := secretbox.Seal(nil, data, &s.nonce, &s.key)
	// write version byte
	if _, err := w.Write([]byte{Version}); err != nil {
		return err
	}
	// write salt
	if _, err := w.Write(s.salt[:]); err != nil {
		return err
	}
	// write nonce
	if _, err := w.Write(s.nonce[:]); err != nil {
		return err
	}
	// write encrypted data
	if _, err := w.Write(enc); err != nil {
		return err
	}
	return nil
}

// Create storage file and write encrypte data to it.
func Create(filename, passphrase string, data []byte) (*State, error) {
	// make sure keyfile does not exist already
	if _, err := os.Stat(filename); err == nil {
		return nil, fmt.Errorf("storage: file '%s' exists already", filename)
	}
	// generate salt
	s := &State{filename: filename}
	if _, err := io.ReadFull(rand.Reader, s.salt[:]); err != nil {
		return nil, err
	}
	// generate nonce
	if _, err := io.ReadFull(rand.Reader, s.nonce[:]); err != nil {
		return nil, err
	}
	// compute derived key from passphrase
	key := argon2.IDKey([]byte(passphrase), s.salt[:], 1, 64*1024, 4, 32)
	copy(s.key[:], key)
	// open file
	fp, err := os.Create(s.filename)
	if err != nil {
		return nil, err
	}
	// sava encrypted data
	if err := s.save(fp, data); err != nil {
		fp.Close()
		os.Remove(fp.Name())
		return nil, err
	}
	if err := fp.Close(); err != nil {
		os.Remove(fp.Name())
		return nil, err
	}
	return s, nil
}

// Open encrypted file and return decrypted data.
func Open(filename, passphrase string) (*State, []byte, error) {
	// open encrypted file
	fp, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer fp.Close()
	s := &State{filename: filename}
	// read version byte
	var version [1]byte
	if _, err := fp.Read(version[:]); err != nil {
		return nil, nil, err
	}
	if version[0] != Version {
		return nil, nil,
			fmt.Errorf("storage: read version %d incompatible with expected version %d",
				version[0], Version)
	}
	// read salt
	if _, err := fp.Read(s.salt[:]); err != nil {
		return nil, nil, err
	}
	// read nonce
	if _, err := fp.Read(s.nonce[:]); err != nil {
		return nil, nil, err
	}
	// derive key
	key := argon2.IDKey([]byte(passphrase), s.salt[:], 1, 64*1024, 4, 32)
	copy(s.key[:], key)
	// read encrypted data
	enc, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, nil, err
	}
	// decrypt data
	data, verify := secretbox.Open(nil, enc, &s.nonce, &s.key)
	if !verify {
		return nil, nil, fmt.Errorf("storage: cannot decrypt '%s'", filename)
	}
	return s, data, nil
}

// Save new data, overwriting old!
func (s *State) Save(data []byte) error {
	tmpfile := s.filename + ".new"
	os.Remove(tmpfile) // ignore error
	// open file
	fp, err := os.Create(tmpfile)
	if err != nil {
		return err
	}
	// sava encrypted data
	if err := s.save(fp, data); err != nil {
		fp.Close()
		os.Remove(fp.Name())
		return err
	}
	// write temp. file
	if err := fp.Close(); err != nil {
		os.Remove(fp.Name())
		return err
	}
	// move temp. file in place
	return os.Rename(tmpfile, s.filename)
}

// Rekey file.
func Rekey(filename, oldPassphrase, newPassphrase string) error {
	_, data, err := Open(filename, oldPassphrase)
	if err != nil {
		return err
	}
	tmpfile := filename + ".new"
	os.Remove(tmpfile) // ignore error
	_, err = Create(tmpfile, newPassphrase, data)
	if err != nil {
		return err
	}
	// move temp. file in place
	return os.Rename(tmpfile, filename)
}
