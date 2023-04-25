package main

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"ser1.net/clapconf"
	"ser1.net/claptrap/v4"
	"ser1.net/kpmenu/kpmenulib"
)

// Version is the version of kpmenu
const Version = "1.5.0"

func main() {
	cc := initializeFlags()
	if cc.Bool("version") {
		fmt.Println(Version)
		os.Exit(0)
	}
	if cc.Bool("help") {
		claptrap.Usage()
		os.Exit(0)
	}
	config := kpmenulib.NewConfiguration()
	if err := loadConfig(cc, config); err != nil {
		log.Fatalf("loading config: %s", err)
		os.Exit(1)
	}

	menu, err := kpmenulib.NewMenu(config)
	if err != nil {
		log.Fatalf("creating menu: %s", err)
		os.Exit(1)
	}
	menu.ReloadConfig = func() error {
		return loadConfig(cc, config)
	}

	// Start client
	if err = kpmenulib.StartClient(); err != nil {
		// Failed to comunicate with server - start server
		err = kpmenulib.StartServer(menu)

		if err != nil {
			log.Fatalf("starting server: %s", err)
			os.Exit(1)
		} else {
			log.Printf("waiting for goroutines to end")
			// Wait for any goroutine (clipboard)
			menu.WaitGroup.Wait()
		}
	}
}

// InitializeFlags prepare cli flags
func initializeFlags() *claptrap.CommandConfig {
	reg := claptrap.Command("kpmenu", "Keepass Menu")

	// Flags
	reg.Add("--daemon", false, "Start kpmenu directly as daemon")
	reg.Add("--version", "-v", false, "Show kpmenu version")
	reg.Add("--autotype", false, "Initiate autotype")
	reg.Add("--quit", "-q", "Exit the daemon if it is running")
	reg.Add("--help", "-h", "Print help and exit")

	// General
	reg.Add("--menu", "-m", kpmenulib.PromptDmenu, "Choose which menu to use")                                          // &c.General.Menu
	reg.Add("--clipboardTool", kpmenulib.ClipboardToolXsel, "Choose which clipboard tool to use")                       // &c.General.ClipboardTool
	reg.Add("--clipboardTimeout", "-c", 15*time.Second, "Timeout of clipboard in seconds (0 = no timeout)")             // &c.General.ClipboardTimeout
	reg.Add("--nocache", "-n", false, "Disable caching of database")                                                    // &c.General.NoCache
	reg.Add("--cacheOneTime", false, "Cache the database only the first time")                                          // &c.General.CacheOneTime
	reg.Add("--cacheTimeout", 60*time.Second, "Timeout of cache in seconds")                                            // &c.General.CacheTimeout
	reg.Add("--nootp", false, "Disable OTP handling")                                                                   // &c.General.NoOTP
	reg.Add("--noautotype", false, "Disable autotype handling")                                                         // &c.General.DisableAutotype
	reg.Add("--autotypealwaysconfirm", false, "Always confirm autotype, even when there's only 1 selection")            // &c.General.AutotypeConfirm
	reg.Add("--autotypeusersel", false, "Prompt for autotype entry instead of trying to detect by active window title") // &c.General.AutotypeNoAuto

	// Executable
	reg.Add("--customPromptPassword", "", "Custom executable for prompt password")                                                          // &c.Executable.CustomPromptPassword
	reg.Add("--customPromptMenu", "", "Custom executable for prompt menu")                                                                  // &c.Executable.CustomPromptMenu
	reg.Add("--customPromptEntries", "", "Custom executable for prompt entries")                                                            // &c.Executable.CustomPromptEntries
	reg.Add("--customPromptFields", "", "Custom executable for prompt fields")                                                              // &c.Executable.CustomPromptFields
	reg.Add("--customClipboardCopy", "", "Custom executable for clipboard copy")                                                            // &c.Executable.CustomClipboardCopy
	reg.Add("--customClipboardPaste", "", "Custom executable for clipboard paste")                                                          // &c.Executable.CustomClipboardPaste
	reg.Add("--customAutotypeWindowID", kpmenulib.AutotypeWindowIdentifier, "Custom executable for identifying active window for autotype") // &c.Executable.CustomAutotypeWindowID
	reg.Add("--customAutotypeTyper", kpmenulib.AutotypeTyper, "Custom executable for autotype typer")                                       // &c.Executable.CustomAutotypeTyper
	reg.Add("--customClipboardClean", "", "Custom executable for clipboard clean")                                                          // &c.Executable.CustomClipboardClean

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

	reg.Parse(nil)
	return reg
}

// LoadConfig loads the configuration into Configuration
func loadConfig(reg *claptrap.CommandConfig, conf *kpmenulib.Configuration) error {
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
	return nil
}
