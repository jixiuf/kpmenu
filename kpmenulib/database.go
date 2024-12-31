package kpmenulib

import (
	"encoding/hex"
	"log"
	"os"
	"strings"

	"github.com/tobischo/gokeepasslib/v3"
)

// Database contains the KeePass database and its entry list
type Database struct {
	Loaded  bool
	Keepass *gokeepasslib.Database
	Entries []Entry
}

// Entry is a container for keepass entry
type Entry struct {
	UUID      gokeepasslib.UUID
	FullEntry gokeepasslib.Entry
}

// NewDatabase initializes the Database struct
func NewDatabase() *Database {
	return &Database{
		Loaded:  false,
		Keepass: gokeepasslib.NewDatabase(),
	}
}

// AddCredentialsToDatabase adds credentials into gokeepasslib credentials struct
func (db *Database) AddCredentialsToDatabase(cfg *Configuration, password string, challenge []byte) (err error) {
	var keyData []byte
	if cfg.Database.KeyFile != "" {
		keyData, err = gokeepasslib.ParseKeyFile(cfg.Database.KeyFile)
	}

	// Get credentials
	if password != "" && keyData != nil {
		// Both password & keyfile
		db.Keepass.Credentials, _ = gokeepasslib.NewPasswordAndKeyDataCredentials(password, keyData)
		log.Printf("credentials: password + keyfile")
	} else if password != "" {
		// Only password
		db.Keepass.Credentials = gokeepasslib.NewPasswordCredentials(password)
		log.Printf("credentials: password")
	} else if keyData != nil {
		// Only keyfile
		// db.Keepass.Credentials, _ = gokeepasslib.NewKeyCredentials(cfg.Database.KeyFile)
		db.Keepass.Credentials, _ = gokeepasslib.NewKeyDataCredentials(keyData)
		log.Printf("credentials: keyfile ")
	}
	if cfg.Database.KeyFileData != "" {
		if fileExists(cfg.Database.KeyFileData) {
			keyFileData, _ := gokeepasslib.ParseKeyFile(cfg.Database.KeyFileData)
			db.Keepass.Credentials.Windows, _ = gokeepasslib.ParseKeyData(keyFileData)
		} else {
			cmd := strings.ReplaceAll(cfg.Database.KeyFileData, "%salt", hex.EncodeToString(challenge))
			cmd = strings.ReplaceAll(cmd, "%database", cfg.Database.Database)
			cmd = strings.ReplaceAll(cmd, "%password", password)
			// ykchalresp -x -2 -H %salt
			keyFileData, err := run("sh", "", "-c", cmd)
			if err != nil {
				log.Printf("credentials: keyFileData: %v", err)
				return err
			}
			keyFileData, _ = hex.DecodeString(string(keyFileData))
			db.Keepass.Credentials.Windows, _ = gokeepasslib.ParseKeyData(keyFileData)
		}
	}
	return
}
func (db *Database) DeocdeDatabase(cfg *Configuration) error {
	// Open database file
	file, err := os.Open(cfg.Database.Database)
	if err == nil {
		err = gokeepasslib.NewDecoder(file).Decode(db.Keepass)
	}
	return err
}

// OpenDatabase decodes the database with the given configuration
func (db *Database) OpenDatabase(cfg *Configuration) error {
	// Open database file
	file, err := os.Open(cfg.Database.Database)
	if err == nil {
		err = gokeepasslib.NewDecoder(file).Decode(db.Keepass)
		if err == nil {
			err = db.Keepass.UnlockProtectedEntries()
		}
	}
	return err
}

// IterateDatabase iterates the database and makes a list of entries
func (db *Database) IterateDatabase() {
	var entries []Entry
	for _, sub := range db.Keepass.Content.Root.Groups {
		entries = append(entries, iterateGroup(sub)...)
	}
	db.Entries = entries
}

func iterateGroup(kpGroup gokeepasslib.Group) []Entry {
	var entries []Entry
	// Get entries of the current group
	for _, kpEntry := range kpGroup.Entries {
		// Insert entry
		if kpEntry.AutoType.DefaultSequence == "" {
			kpEntry.AutoType.DefaultSequence = kpGroup.DefaultAutoTypeSequence
		}

		entries = append(entries, Entry{
			UUID:      kpEntry.UUID,
			FullEntry: kpEntry,
		})
		//(*entries)[uuid] = Entry{FullEntry: kpEntry}
	}

	// Continue to iterate subgroups
	for _, sub := range kpGroup.Groups {
		entries = append(entries, iterateGroup(sub)...)
	}
	return entries
}
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
