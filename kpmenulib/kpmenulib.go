package kpmenulib

import (
	"errors"
	"log"
	"os/exec"
)

func validateConfig(config *Configuration) error {
	// Check if database has been selected
	if config.Database.Database == "" {
		// Database not found
		return errors.New("you must select a database with -d or via config")
	}

	// Check if rofi is installed
	if config.General.Menu == PromptRofi {
		cmd := exec.Command("which", "rofi")
		err := cmd.Run()
		if err != nil {
			log.Printf("rofi not found, using dmenu")
			config.General.Menu = PromptDmenu
		}
	} else if config.General.Menu == PromptWofi {
		cmd := exec.Command("which", "wofi")
		err := cmd.Run()
		if err != nil {
			log.Printf("wofi not found, using dmenu")
			config.General.Menu = PromptDmenu
		}
	} else if config.General.Menu != PromptDmenu && config.General.Menu != PromptCustom {
		return errors.New("invalid menu option, exiting")
	}

	if config.General.Menu == PromptDmenu {
		// Check if dmenu is installed
		cmd := exec.Command("which", "dmenu")
		err := cmd.Run()
		if err != nil {
			return errors.New("dmenu not found, exiting")
		}
	}

	if config.General.ClipboardTool == ClipboardToolWlclipboard {
		// Check if wl-clipboard is installed
		cmd := exec.Command("which", "wl-copy")
		err := cmd.Run()
		if err != nil {
			return errors.New("wl-clipboard not found, exiting")
		}
	} else if config.General.ClipboardTool == ClipboardToolXsel {
		// Check if xsel is installed
		cmd := exec.Command("which", "xsel")
		err := cmd.Run()
		if err != nil {
			return errors.New("xsel not found, exiting")
		}
	} else if config.General.ClipboardTool == ClipboardToolCustom {
		// Check if CustomClipboardCopy, CustomClipboardPaste and CustomClipboardClean are set
		if config.Executable.CustomClipboardCopy == "" {
			return errors.New("when clipboardTool is set to custom, CustomClipboardCopy must be set")
		}
		if config.Executable.CustomClipboardPaste == "" {
			return errors.New("when clipboardTool is set to custom, CustomClipboardPaste must be set")
		}
		if config.Executable.CustomClipboardClean == "" {
			return errors.New("when clipboardTool is set to custom, CustomClipboardClean must be set")
		}
	}
	return nil
}
