package kpmenulib

import (
	"fmt"
	"time"
)

// Configuration is the main structure of kpmenu config
type Configuration struct {
	General    ConfigurationGeneral
	Executable ConfigurationExecutable
	Style      ConfigurationStyle
	Database   ConfigurationDatabase
	Flags      Flags
}

// ConfigurationGeneral is the sub-structure of the configuration related to general kpmenu settings
type ConfigurationGeneral struct {
	Menu             string        // Which menu to use
	ClipboardTool    string        // Clipboard tool to use
	ClipboardTimeout time.Duration // Clipboard timeout before clean it
	NoCache          bool          // Flag to do not cache master password
	CacheOneTime     bool          // Cache the password only the first time you write it
	CacheTimeout     time.Duration // Timeout of cache
	NoOTP            bool          // Flag to do not handle OTPs
	DisableAutotype  bool          // Disable autotype
	AutotypeConfirm  bool          // User must always confirm
	AutotypeNoAuto   bool          // Always prompt user to select the entry to autotype
}

// ConfigurationExecutable is the sub-structure of the configuration related to tools executed by kpmenu
type ConfigurationExecutable struct {
	CustomPromptPassword   string // Custom executable for prompt password
	CustomPromptMenu       string // Custom executable for prompt menu
	CustomPromptEntries    string // Custom executable for prompt entries
	CustomPromptFields     string // Custom executable for prompt fields
	CustomClipboardCopy    string // Custom executable for clipboard copy
	CustomClipboardPaste   string // Custom executable for clipboard paste
	CustomClipboardClean   string // Custom executable for clipboard clean
	CustomAutotypeWindowID string // Custom executable for fetching title of active window
	CustomAutotypeTyper    string // Custom executable for typing results
}

// ConfigurationStyle is the sub-structure of the configuration related to style of dmenu
type ConfigurationStyle struct {
	PasswordBackground string
	TextPassword       string
	TextMenu           string
	TextEntry          string
	TextField          string
	FormatEntry        string
	ArgsPassword       string
	ArgsMenu           string
	ArgsEntry          string
	ArgsField          string
}

// ConfigurationDatabase is the sub-structure of the configuration related to database settings
type ConfigurationDatabase struct {
	Database        string
	KeyFile         string
	Password        string
	FieldOrder      string
	FillOtherFields bool
	FillBlacklist   string
}

// Flags is the sub-structure of the configuration used to handle flags that aren't into the config file
type Flags struct {
	Daemon   bool
	Version  bool
	Autotype bool
}

// Menu tools used for prompts
const (
	PromptDmenu  = "dmenu"
	PromptRofi   = "rofi"
	PromptWofi   = "wofi"
	PromptCustom = "custom"
)

// Clipboard tools used for clipboard manager
const (
	ClipboardToolXsel        = "xsel"
	ClipboardToolWlclipboard = "wl-clipboard"
	ClipboardToolCustom      = "custom"
)

// Autotype default helpers
const (
	AutotypeWindowIdentifier = "quasiauto -title"
	AutotypeTyper            = "quasiauto"
)

// NewConfiguration initializes a new Configuration pointer
func NewConfiguration() *Configuration {
	return &Configuration{
		General: ConfigurationGeneral{
			Menu:             PromptDmenu,
			ClipboardTool:    ClipboardToolXsel,
			ClipboardTimeout: 15 * time.Second,
			CacheTimeout:     60 * time.Second,
		},
		Style: ConfigurationStyle{
			PasswordBackground: "black",
			TextPassword:       "Password",
			TextMenu:           "Select",
			TextEntry:          "Entry",
			TextField:          "Field",
			FormatEntry:        "{Title} - {UserName}",
		},
		Database: ConfigurationDatabase{
			FieldOrder:      "Password UserName URL",
			FillOtherFields: true,
		},
		Executable: ConfigurationExecutable{
			CustomAutotypeWindowID: AutotypeWindowIdentifier,
			CustomAutotypeTyper:    AutotypeTyper,
		},
	}
}

// ErrParseConfiguration is the error return if the configuration loading fails
type ErrParseConfiguration struct {
	Message       string
	OriginalError error
}

// NewErrorParseConfiguration initializes the error
func NewErrorParseConfiguration(message string, err error) ErrParseConfiguration {
	return ErrParseConfiguration{
		Message:       message,
		OriginalError: err,
	}
}

func (err ErrParseConfiguration) Error() string {
	return fmt.Sprintf(err.Message, err.OriginalError)
}
