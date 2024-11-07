package kpmenulib

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"ser1.net/clapconf"

	"ser1.net/claptrap/v4"
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

// InitializeFlags prepare cli flags
func InitializeFlags(args []string) *claptrap.CommandConfig {
	reg := claptrap.Command("kpmenu", "Keepass Menu")

	// Flags
	reg.Add("--daemon", false, "Start kpmenu directly as daemon")
	reg.Add("--version", "-v", false, "Show kpmenu version")
	reg.Add("--autotype", false, "Initiate autotype")
	reg.Add("--quit", "-q", "Exit the daemon if it is running")
	reg.Add("--help", "-h", "Print help and exit")

	// General
	reg.Add("--menu", "-m", PromptDmenu, "Choose which menu to use")                                                   // &c.General.Menu
	reg.Add("--clipboardTool", ClipboardToolXsel, "Choose which clipboard tool to use")                                // &c.General.ClipboardTool
	reg.Add("--clipboardTimeout", "-c", 15*time.Second, "Timeout of clipboard in seconds (0 = no timeout)")            // &c.General.ClipboardTimeout
	reg.Add("--nocache", "-n", false, "Disable caching of database")                                                   // &c.General.NoCache
	reg.Add("--cacheOneTime", false, "Cache the database only the first time")                                         // &c.General.CacheOneTime
	reg.Add("--cacheTimeout", 60*time.Second, "Timeout of cache in seconds")                                           // &c.General.CacheTimeout
	reg.Add("--nootp", false, "Disable OTP handling")                                                                  // &c.General.NoOTP
	reg.Add("--noautotype", false, "Disable autotype handling")                                                        // &c.General.DisableAutotype
	reg.Add("--autotypealwaysconfirm", false, "Always confirm autotype, even when there's only 1 selection")           // &c.General.AutotypeConfirm
	reg.Add("--autotypeNoAuto", false, "Prompt for autotype entry instead of trying to detect by active window title") // &c.General.AutotypeNoAuto

	// Executable
	reg.Add("--customPromptPassword", "", "Custom executable for prompt password")                                                // &c.Executable.CustomPromptPassword
	reg.Add("--customPromptMenu", "", "Custom executable for prompt menu")                                                        // &c.Executable.CustomPromptMenu
	reg.Add("--customPromptEntries", "", "Custom executable for prompt entries")                                                  // &c.Executable.CustomPromptEntries
	reg.Add("--customPromptFields", "", "Custom executable for prompt fields")                                                    // &c.Executable.CustomPromptFields
	reg.Add("--customClipboardCopy", "", "Custom executable for clipboard copy")                                                  // &c.Executable.CustomClipboardCopy
	reg.Add("--customClipboardPaste", "", "Custom executable for clipboard paste")                                                // &c.Executable.CustomClipboardPaste
	reg.Add("--customAutotypeWindowID", AutotypeWindowIdentifier, "Custom executable for identifying active window for autotype") // &c.Executable.CustomAutotypeWindowID
	reg.Add("--customAutotypeTyper", AutotypeTyper, "Custom executable for autotype typer")                                       // &c.Executable.CustomAutotypeTyper
	reg.Add("--customClipboardClean", "", "Custom executable for clipboard clean")                                                // &c.Executable.CustomClipboardClean

	// Style
	reg.Add("--passwordBackground", "black", "Color of dmenu background and text for password selection, used to hide password typing") // &c.Style.PasswordBackground
	reg.Add("--textPassword", "Password", "Label for password selection")                                                               // &c.Style.TextPassword
	reg.Add("--textMenu", "Select", "Label for menu selection")                                                                         // &c.Style.TextMenu
	reg.Add("--textEntry", "Entry", "Label for entry selection")                                                                        // &c.Style.TextEntry
	reg.Add("--textField", "Field", "Label for field selection")                                                                        // &c.Style.TextField
	reg.Add("--argsPassword", "", "Additional arguments for dmenu at password selection, separated by a space")                         // &c.Style.ArgsPassword
	reg.Add("--argsMenu", "", "Additional arguments for dmenu at menu selection, separated by a space")                                 // &c.Style.ArgsMenu
	reg.Add("--argsEntry", "", "Additional arguments for dmenu at entry selection, separated by a space")                               // &c.Style.ArgsEntry
	reg.Add("--argsField", "", "Additional arguments for dmenu at field selection, separated by a space")                               // &c.Style.ArgsField
	reg.Add("--formatEntry", "{Title} - {UserName}", "Template for the entry list")

	// Database
	reg.Add("--database", "-d", "", "Path to the KeePass database")                                       // &c.Database.Database
	reg.Add("--keyfile", "-k", "", "Path to the database keyfile")                                        // &c.Database.KeyFile
	reg.Add("--password", "-p", "", "Password of the database")                                           // &c.Database.Password
	reg.Add("--fieldOrder", "Password UserName URL", "String order of fields to show on field selection") // &c.Database.FieldOrder
	reg.Add("--fillOtherFields", false, "Enable fill of remaining fields")                                // &c.Database.FillOtherFields
	reg.Add("--fillBlacklist", "", "String of blacklisted fields that won't be shown")                    // &c.Database.FillBlacklist

	reg.Parse(args)
	return reg
}

// LoadConfig loads the configuration into Configuration
func LoadConfig(reg *claptrap.CommandConfig, conf *Configuration) error {
	// FIXME might have to manually load the config, b/c of the differences in config serialization
	err := clapconf.LoadConfig("")
	if err != nil {
		log.Print("If upgrading from AlessioDP/kpmenu, the configuration file has changed.")
		log.Print("Remove all of the section headings (e.g. '[general]'), and camelCase the")
		log.Print("fields (e.g. 'ClipboardTimeout' -> 'clipboardTimeout')")
		log.Print("Add units to clipboardTimeout and cacheTimeout, e.g 'clipboardTimeout=30s'")
		return err
	}

	setAll := func(it reflect.Value, typ reflect.Type) {
		item := it.Elem()
		duration := reflect.ValueOf(time.Second).Kind()
		for _, f := range reflect.VisibleFields(typ) {
			lower := strings.ToLower(f.Name)
			argName := fmt.Sprintf("%c%s", lower[0], f.Name[1:])
			value := item.FieldByIndex(f.Index)
			switch f.Type.Kind() {
			case reflect.Bool:
				value.Set(reflect.ValueOf(reg.Bool(argName)))
			case reflect.String:
				value.SetString(reg.String(argName))
			case reflect.Int:
				value.Set(reflect.ValueOf(reg.Int(argName)))
			case duration:
				value.Set(reflect.ValueOf(reg.Duration(argName)))
			}
		}
	}
	setAll(reflect.ValueOf(&conf.General), reflect.TypeOf(conf.General))
	setAll(reflect.ValueOf(&conf.Style), reflect.TypeOf(conf.Style))
	setAll(reflect.ValueOf(&conf.Database), reflect.TypeOf(conf.Database))
	setAll(reflect.ValueOf(&conf.Executable), reflect.TypeOf(conf.Executable))
	setAll(reflect.ValueOf(&conf.Flags), reflect.TypeOf(conf.Flags))
	return nil
}
