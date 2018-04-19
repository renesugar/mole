// Package config defines all (persistent) data of a Mole instance.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
)

// Settings dfines the global settings of a Mole instance.
type Settings struct {
	Resource string // generated XMPP client resource (e.g., 'mole-VX9Nzrq_WV-iyI6SF7KskA')
}

// Account defines a XMPP account.
type Account struct {
	Username string // own JID
	Password string
	Contact  string // TODO: remove
	// Hostname string // optional
	// Port     int    // optional
}

// Contact defines a XMPP contact.
type Contact struct {
	Remote string // JID of contact
	Local  string // own JID, corresponds to an account
}

// A Hill contains all data of a Mole instance.
type Hill struct {
	Settings Settings
	Accounts []Account
	Contacts []Contact
}

// genResource generates a unique XMPP client resource string.
func genResource() (string, error) {
	pass := make([]byte, 16) // 128-bit
	if _, err := io.ReadFull(rand.Reader, pass[:]); err != nil {
		return "", err
	}
	return "mole-" + base64.RawURLEncoding.EncodeToString(pass[:]), nil
}

// NewHill generates a new Hill with default settings.
func NewHill() (*Hill, error) {
	var h Hill
	resource, err := genResource()
	if err != nil {
		return nil, err
	}
	h.Settings.Resource = resource
	return &h, nil
}

// Marshal a Hill without indentation.
func (h *Hill) Marshal() []byte {
	jsn, err := json.Marshal(h)
	if err != nil {
		panic(err) // should never happen
	}
	return jsn
}

// MarshalIndent marshalls a Hill with indentation.
func (h *Hill) MarshalIndent() []byte {
	jsn, err := json.MarshalIndent(h, "", "    ")
	if err != nil {
		panic(err) // should never happen
	}
	return jsn
}

// Unmarshal data into a Hill.
func Unmarshal(data []byte) (*Hill, error) {
	var h Hill
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

func (h *Hill) LastAccount() *Account {
	if h.Accounts == nil || len(h.Accounts) == 0 {
		return nil
	}
	return &h.Accounts[len(h.Accounts)-1]
}
