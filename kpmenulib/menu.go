package kpmenulib

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// Menu is the main structure of kpmenu
type Menu struct {
	CacheStart    time.Time       // Cache start time
	CliArguments  []string        // Arguments of kpmenu
	Configuration *Configuration  // Configuration of kpmenu
	Database      *Database       // Database
	WaitGroup     *sync.WaitGroup // WaitGroup used for goroutines
	ReloadConfig  func() error    // Call-back to update configuration options
}

// NewMenu initializes a Menu struct
func NewMenu(config *Configuration) (*Menu, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	menu := Menu{
		CliArguments:  os.Args[1:],
		Configuration: config,
		Database:      NewDatabase(),
		WaitGroup:     new(sync.WaitGroup),
	}

	// Set start cache time, if not a daemon
	if !config.Flags.Daemon && !config.General.NoCache {
		menu.CacheStart = time.Now()
	}

	return &menu, nil
}

// Execute is the function used to open the database (if necessary) and open the menu
// returns true if the program should exit
func (menu *Menu) Execute() bool {
	// Open database
	if menu.Database.Loaded == false {
		if err := menu.OpenDatabase(); err != nil {
			log.Print(err)
			return err.Fatal
		}
	} else if menu.Configuration.Flags.Daemon {
		// reload Database for next time
		defer menu.OpenDatabase()
	}

	if !menu.Configuration.General.DisableAutotype && menu.Configuration.Flags.Autotype {
		if err := PromptAutotype(menu); err.Error != nil {
			log.Print(err.Error)
		}
		return false
	}

	// Open menu
	if err := menu.OpenMenu(); err != nil {
		log.Print(err)
		return err.Fatal
	}

	// Non-fatal exit
	return false
}

// Show checks if the database configuration is changed, if so it will re-open the database
// returns true if the program should exit
func (menu *Menu) Show() bool {
	// Be sure that the database configuration is the same, otherwise a Run is necessary
	copiedDatabase := menu.Configuration.Database

	// If something related to the database is changed we must re-open it, or exit true
	if copiedDatabase.Database != menu.Configuration.Database.Database ||
		copiedDatabase.KeyFile != menu.Configuration.Database.KeyFile ||
		copiedDatabase.Password != menu.Configuration.Database.Password {
		menu.Database.Loaded = false
		log.Printf("database configuration is changed, re-opening the database")
	}

	// Check if the cache is not timed out, if not a daemon
	if !menu.Configuration.Flags.Daemon {
		if menu.Configuration.General.NoCache {
			// Cache disabled
			menu.Database.Loaded = false
			log.Printf("no cache flag is set, re-opening the database")
		} else if (menu.CacheStart == time.Time{}) {
			// Cache enabled via client call
			menu.Database.Loaded = false
			log.Printf("cache start time not set, re-opening the database")
		} else {
			// Cache exists
			difference := time.Now().Sub(menu.CacheStart)
			if difference < menu.Configuration.General.CacheTimeout {
				// Cache is valid
				if !menu.Configuration.General.CacheOneTime {
					// Set new cache start if cache one time is false
					menu.CacheStart = time.Now()
				}
			} else {
				// Cache timed out
				menu.Database.Loaded = false
				log.Printf("cache timed out, re-opening the database")
			}
		}
	}

	return menu.Execute()
}

// OpenDatabase asks for password and populates the database
func (m *Menu) OpenDatabase() *ErrorDatabase {
	// Check if there is already a password/key set
	if !m.Database.Loaded {
		// Get password from config otherwise ask for it
		password := m.Configuration.Database.Password
		if password == "" {
			// Get password from user
			pw, err := PromptPassword(m)
			if !err.Cancelled {
				if err.Error != nil {
					return NewErrorDatabase("failed to get password from dmenu: %s", err.Error, true)
				}
			} else {
				// Exit because cancelled
				return NewErrorDatabase("exiting because user cancelled password prompt", nil, true)
			}
			password = pw
		}

		// Add credentials into the database
		m.Database.AddCredentialsToDatabase(m.Configuration, password)
	}

	// Open database
	if err := m.Database.OpenDatabase(m.Configuration); err != nil {
		return NewErrorDatabase("failed to open database: %s", err, true)
	}

	// Get entries of database
	m.Database.IterateDatabase()

	// Set database as loaded
	m.Database.Loaded = true

	return nil
}

// OpenMenu executes dmenu to interface the user with the database
func (m *Menu) OpenMenu() *ErrorDatabase {
	// Prompt for menu selection
	selectedMenu, err := PromptMenu(m)
	if err.Cancelled {
		if err.Error != nil {
			return NewErrorDatabase("failed to select menu item: %s", err.Error, false)
		}
		// Cancelled
		return NewErrorDatabase("", nil, false)
	}
	switch selectedMenu {
	case MenuShow:
		return m.entrySelection()
	case MenuReload:
		log.Printf("reloading database")
		if err := m.OpenDatabase(); err != nil {
			return err
		}
		return m.OpenMenu()
	case MenuExit:
		m.Database.Loaded = false
		return NewErrorDatabase("exiting", nil, true)
	}
	return nil
}

func (m *Menu) entrySelection() *ErrorDatabase {
	// Prompt for entry selection
	selectedEntry, err := PromptEntries(m)
	if err.Cancelled {
		if err.Error != nil {
			return NewErrorDatabase("failed to select entry: %s", err.Error, false)
		}
		// Cancelled
		return NewErrorDatabase("", nil, false)
	}
	if selectedEntry == nil {
		// Entry not found
		return NewErrorDatabase("selected entry not found", nil, false)
	}

	// Prompt for field selection
	fieldValue, err := PromptFields(m, selectedEntry)
	if err.Cancelled {
		if err.Error != nil {
			return NewErrorDatabase("failed to select field: %s", err.Error, false)
		}
		// Cancelled
		return NewErrorDatabase("", nil, false)
	}
	if fieldValue == "" {
		// Field not found
		return NewErrorDatabase("selected field not found", nil, false)
	}

	// Copy to clipboard
	if err := CopyToClipboard(m, fieldValue); err != nil {
		return NewErrorDatabase("failed to use clipboard manager to update clipboard: %s", err, true)
	}
	log.Printf("copied field into the clipboard")

	// Clean clipboard (goroutine)
	CleanClipboard(m, fieldValue)
	return nil
}

// ErrorDatabase is an error that can be fatal or non-fatal
type ErrorDatabase struct {
	Message       string
	OriginalError error
	Fatal         bool
}

// NewErrorDatabase makes an ErrorDatabase
func NewErrorDatabase(message string, err error, fatal bool) *ErrorDatabase {
	return &ErrorDatabase{
		Message:       message,
		OriginalError: err,
		Fatal:         fatal,
	}
}

func (err *ErrorDatabase) String() string {
	if err.OriginalError != nil {
		return fmt.Sprintf(err.Message, err.OriginalError)
	}
	return err.Message
}
