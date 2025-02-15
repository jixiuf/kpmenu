package kpmenulib

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/google/shlex"
	"github.com/tobischo/gokeepasslib/v3"
)

// MenuSelection is an enum used for prompt menu selection
type MenuSelection int

// MenuSelections enum values
const (
	MenuShow   = MenuSelection(iota) // Show entries
	MenuReload                       // Reload database
	MenuExit                         // Exit
)

var menuSelections = [...]string{
	"Show entries",
	"Reload database",
	"Exit",
}

type entryItem struct {
	Title string
	Entry *Entry
}

// ErrorPrompt is a structure that handle an error of dmenu/rofi
type ErrorPrompt struct {
	Cancelled bool
	KbCustom1 bool
	KbCustom2 bool
	KbCustom3 bool
	KbCustom4 bool
	KbCustom5 bool
	Error     error
}

// PromptPassword executes dmenu to ask for database password
// Returns the written password
func PromptPassword(menu *Menu) (string, ErrorPrompt) {
	// Prepare dmenu/rofi
	command, err := getCommand(menu, menu.Configuration.Style.TextPassword, true, menu.Configuration.Executable.CustomPromptPassword)
	ep := ErrorPrompt{}
	if err != ep {
		return "", err
	}

	// Add custom arguments
	if menu.Configuration.Style.ArgsPassword != "" {
		command = append(command, strings.Split(menu.Configuration.Style.ArgsPassword, " ")...)
	}

	// Execute prompt
	return executePrompt(command, nil)
}

// PromptMenu executes dmenu to ask for menu selection
// Returns the MenuSelection chosen
func PromptMenu(menu *Menu) (MenuSelection, ErrorPrompt) {
	var selection MenuSelection
	var input strings.Builder

	// Prepare dmenu/rofi
	command, err := getCommand(menu, menu.Configuration.Style.TextMenu, false, menu.Configuration.Executable.CustomPromptMenu)
	ep := ErrorPrompt{}
	if err != ep {
		return selection, err
	}

	// Add custom arguments
	if menu.Configuration.Style.ArgsMenu != "" {
		command = append(command, strings.Split(menu.Configuration.Style.ArgsMenu, " ")...)
	}

	// Prepare input (dmenu items)
	for _, e := range menuSelections {
		input.WriteString(e + "\n")
	}

	// Execute prompt
	result, err := executePrompt(command, strings.NewReader(input.String()))
	if err.Error == nil && !err.Cancelled {
		// Get selected menu item
		for ind, sel := range menuSelections {
			// Match for entry title and selected entry
			if sel == result {
				selection = MenuSelection(ind)
				break
			}
		}
	}
	return selection, err
}

// PromptEntries executes dmenu to ask for an entry selection
// Returns the selected entry
func PromptEntries(menu *Menu) (*Entry, ErrorPrompt) {
	var entry Entry
	var input strings.Builder

	// Prepare autotype command
	command, erp := getCommand(menu, menu.Configuration.Style.TextEntry, false, menu.Configuration.Executable.CustomPromptEntries)
	ep := ErrorPrompt{}
	if erp != ep {
		return &entry, erp
	}

	// Add custom arguments
	if menu.Configuration.Style.ArgsEntry != "" {
		command = append(command, strings.Split(menu.Configuration.Style.ArgsEntry, " ")...)
	}

	// Prepare a list of entries
	// Identified by the formatted title and the entry pointer
	var listEntries []entryItem
	reg, err := regexp.Compile(`{[a-zA-Z]+\}`)
	if err != nil {
		return &entry, ErrorPrompt{
			Cancelled: false,
			Error:     err,
		}
	}
	for i, e := range menu.Database.Entries {
		// Format entry
		title := menu.Configuration.Style.FormatEntry
		matches := reg.FindAllString(title, -1)

		// Replace every match
		for _, match := range matches {
			valueType := match[1 : len(match)-1] // Removes { and }
			value := ""                          // By default empty value
			vd := e.FullEntry.GetContent(valueType)
			if vd != "" {
				value = vd
			}
			title = strings.Replace(title, match, value, -1)
		}
		// Be sure to point on the right entry, do not point to the local e
		listEntries = append(listEntries, entryItem{Title: title, Entry: &menu.Database.Entries[i]})
	}

	// Prepare input (dmenu items)
	for _, e := range listEntries {
		input.WriteString(e.Title + "\n")
	}

	// Execute prompt
	result, errPrompt := executePrompt(command, strings.NewReader(input.String()))
	if errPrompt.Error == nil && !errPrompt.Cancelled {
		// Get selected entry
		for _, e := range listEntries {
			if e.Title == result {
				entry = *e.Entry
				break
			}
		}
	}
	return &entry, errPrompt
}

// PromptFields executes dmenu to ask for a field selection
// Returns the selected field value as string
func PromptFields(menu *Menu, entry *Entry) (string, ErrorPrompt) {
	var value string
	var input strings.Builder

	// Prepare autotype command
	command, erp := getCommand(menu, menu.Configuration.Style.TextEntry, false, menu.Configuration.Executable.CustomPromptFields)
	ep := ErrorPrompt{}
	if erp != ep {
		return value, erp
	}

	// Add custom arguments
	if menu.Configuration.Style.ArgsField != "" {
		command = append(command, strings.Split(menu.Configuration.Style.ArgsField, " ")...)
	}

	fields := []string{}
	fieldsOrder := strings.Split(menu.Configuration.Database.FieldOrder, " ")

	// Populate ordered fields
	for _, f := range fieldsOrder {
		if entry.FullEntry.GetContent(f) != "" {
			fields = append(fields, f)
		}
	}

	var hasOTP bool
	// Populate with filling fields
	if menu.Configuration.Database.FillOtherFields {
		blacklistFields := strings.Split(menu.Configuration.Database.FillBlacklist, " ")

		for _, v := range entry.FullEntry.Values {
			if !menu.Configuration.General.NoOTP && (v.Key == OTP || v.Key == TOTPSEED) {
				hasOTP = true
				continue
			}
			if !contains(fields, v.Key) && !contains(blacklistFields, v.Key) {
				if v.Value.Content != "" {
					fields = append(fields, v.Key)
				}
			}
		}
	}

	// Prepare input (dmenu items)
	const GenerateOTP = "Generate OTP"
	for _, f := range fields {
		input.WriteString(f + "\n")
	}
	if hasOTP {
		input.WriteString(GenerateOTP + "\n")
	}

	// Execute prompt
	result, err := executePrompt(command, strings.NewReader(input.String()))
	if err.Error == nil && !err.Cancelled {
		if result == GenerateOTP {
			var ev error
			value, ev = CreateOTP(entry.FullEntry, time.Now().Unix())
			if ev != nil {
				err.Cancelled = true
				err.Error = fmt.Errorf("failed to create otp: %s", ev)
				return value, err
			}
			return value, err
		}
		// Check that the result is valid
		if contains(fields, result) {
			// Get field value
			for _, v := range entry.FullEntry.Values {
				if result == v.Key {
					value = v.Value.Content
					break
				}
			}
		}
	}
	return value, err
}

func PromptChoose(menu *Menu, items []string) (int, ErrorPrompt) {
	var input strings.Builder

	// Prepare autotype command
	command, erp := getCommand(menu, menu.Configuration.Style.TextEntry, false, menu.Configuration.Executable.CustomPromptFields)
	ep := ErrorPrompt{}
	if erp != ep {
		return -1, erp
	}

	// Prepare input (dmenu items)
	for _, e := range items {
		input.WriteString(e + "\n")
	}

	// Execute prompt
	result, err := executePrompt(command, strings.NewReader(input.String()))
	if err.Error == nil && !err.Cancelled {
		// Ensures selection is one of the items
		for i, sel := range items {
			// Match for entry title and selected entry
			if result == sel {
				return i, err
			}
		}
	}
	return -1, err
}

// PromptAutotype executes an external application to select an entry and then
// runs an autotype program with the entry's data.
//
// Field data is sent to the autotype child process on STDIN as TSV data. The
// first row is the key sequence.
//
//	{USERNAME}{TAB}{PASSWORD}{ENTER}
//	key <TAB> value <CR>
//
// If Keepass attributes for autotype exist for the record, they're used. If
//
//	they do not exist, username & password are used; if there is no default key
//
// sequence,  `{username}{tab}{password}{tab}{totp}{enter}` is used if OTP is
// set and OTP  is not disabled, or `{USERNAME}{TAB}{PASSWORD}{ENTER}`
// otherwise. The key  sequence is parsed, and only the key/values defined in
// the sequence are sent. It is incumbent on the autotype callee to parse the
// sequence for meta entries such as DELAY and special characters.
//
// STDIN is closed when all fields have been sent.
//
// [Keepass match rules](https://keepass.info/help/base/autotype.html) are, in
// order of precedence:
// 1. Association string as a regexp (if the string is wrapped in `//`)
// 2. Association string as a simple glob
// 3. Entry title as a subset of the window title
//
// If multiple matches are found, the user is prompted to select one.
//
// If `--autotypenoauto` is set, the user will *always* be prompted run autotype, or cancel.
//
// `--noautotype` is handled by the caller -- this function does not perform that check.
func PromptAutotype(menu *Menu, out *PacketResp) ErrorPrompt {
	var entry *Entry
	// The rule for keepass(es) key sequence selection is:
	//    Assoc > Configured entry default > appl. default
	var keySeq = menu.Configuration.General.AutotypeSequence
	var errPrompt ErrorPrompt
	if menu.Configuration.General.AutotypeNoAuto {
		entry, errPrompt = PromptEntries(menu)
		if entry == nil || errPrompt.Cancelled {
			errPrompt.Cancelled = true
			errPrompt.Error = fmt.Errorf("user cancelled")
			return errPrompt
		}

		// Try to guess the key sequence
		if keySeq == "" {
			keySeq = entry.FullEntry.AutoType.DefaultSequence
		}

		if keySeq == "" {
			if entry.FullEntry.AutoType.Associations != nil {
				for _, assoc := range entry.FullEntry.AutoType.Associations {
					if assoc.KeystrokeSequence != "" {
						keySeq = assoc.KeystrokeSequence
						break
					}
				}
			}
		}
	} else {
		entry, keySeq, errPrompt = identifyWindow(menu)
		if entry == nil || errPrompt.Cancelled {
			errPrompt.Cancelled = true
			errPrompt.Error = fmt.Errorf("no entry matched")
			return errPrompt
		}
	}

	if keySeq == "" {
		keySeq = "{USERNAME}{TAB}{PASSWORD}{ENTER}"
	}

	fe := entry.FullEntry
	var input strings.Builder
	input.WriteString(keySeq)
	input.WriteString("\n")
	seq := NewSequence()
	seq.Parse(keySeq)
	rvp := make(Pairs)

	for _, k := range seq.SeqEntries {
		if k.Type == FIELD {
			var value string
			if k.Token == "TOTP" {
				// If the sequence asks for TOTP but the user has disabled it
				// write a dummy code. **Not** writing it would break the
				// sequence, which is ordered.
				if menu.Configuration.General.NoOTP {
					value = "000000"
				} else {
					var err error
					value, err = CreateOTP(fe, time.Now().Unix())
					if err != nil {
						errPrompt.Cancelled = true
						errPrompt.Error = fmt.Errorf("failed to create otp: %s", err)
						return errPrompt
					}
				}
			} else {
				value = getContent(fe, k.Token)
			}
			input.WriteString(k.Token)
			input.WriteString("\t")
			input.WriteString(value)
			input.WriteString("\n")
			rvp[k.Token] = value
		} else {
			rvp[k.Token] = k.Token
		}
	}
	seq.Keylag = 500 // ms
	switch menu.Configuration.Executable.CustomAutotypeTyper {
	case "":
		seq.Exec(rvp, Robot{})
	case "dotool", "dotoolc":
		seq.Exec(rvp, Dotool{
			cmd: menu.Configuration.Executable.CustomAutotypeTyper,
			lag: seq.Keylag,
		})
	case "echo":
		seq.Exec(rvp, &Echo{
			out: out,
			lag: 0,
		})
	default:
		command := strings.Split(menu.Configuration.Executable.CustomAutotypeTyper, " ")
		_, errPrompt = executePrompt(command, strings.NewReader(input.String()))
	}

	return errPrompt
}

func executePrompt(command []string, input *strings.Reader) (result string, errorPrompt ErrorPrompt) {
	var out bytes.Buffer
	var outErr bytes.Buffer
	var cmd *exec.Cmd
	if len(command) == 0 {
		errorPrompt.Cancelled = true
		errorPrompt.Error = fmt.Errorf("the custom prompt command is empty")
		return
	} else if len(command) == 1 {
		cmd = exec.Command(command[0])
	} else {
		cmd = exec.Command(command[0], command[1:]...)
	}

	// Set stdout to out var
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	if input != nil {
		cmd.Stdin = input
	}

	// Run exec
	err := cmd.Run()
	if err != nil {
		if outErr.String() != "" {
			errorPrompt.Error = fmt.Errorf(
				"the command %s returned %s: %s",
				command,
				err,
				outErr.String(),
			)
		} else {
			switch err.Error() {
			case "exit status 10":
				errorPrompt.KbCustom1 = true
			case "exit status 11":
				errorPrompt.KbCustom2 = true
			case "exit status 12":
				errorPrompt.KbCustom3 = true
			case "exit status 13":
				errorPrompt.KbCustom4 = true
			case "exit status 14":
				errorPrompt.KbCustom5 = true

			default:
				errorPrompt.Cancelled = true
			}
		}
	}
	// Trim output
	result = strings.TrimRight(out.String(), "\n")
	return
}

func (el MenuSelection) String() string {
	return menuSelections[el]
}

func getCommand(menu *Menu, style string, pass bool, custom string) ([]string, ErrorPrompt) {
	var command []string
	switch menu.Configuration.General.Menu {
	case "rofi":
		command = []string{
			"rofi",
			"-i",
			"-dmenu",
			"-p", style,
			"-mesg", "Alt-1: type user/passwd, Alt-2: type user, Alt-3: type TOTP, Alt-4:type passwd&RET Alt-5:URL ",
		}
		if pass {
			command = append(command, "-password")
		}
	case "wofi":
		command = []string{
			"wofi",
			"-i",
			"-d",
			"-p", style,
		}
		if pass {
			command = append(command, "--password")
		}
	case "dmenu":
		command = []string{
			"dmenu",
			"-i",
			"-p", style,
		}
		if pass {
			command = append(command, []string{
				"-nb", menu.Configuration.Style.PasswordBackground,
				"-nf", menu.Configuration.Style.PasswordBackground,
			}...)
		}
	case "custom":
		var err error
		command, err = shlex.Split(custom)
		if err != nil {
			var errorPrompt ErrorPrompt
			errorPrompt.Cancelled = true
			errorPrompt.Error = fmt.Errorf("failed to parse custom prompt, exiting")
			return []string{}, errorPrompt
		}
	}
	return command, ErrorPrompt{}
}

func identifyWindow(menu *Menu) (*Entry, string, ErrorPrompt) {
	// Prepare autotype command
	var activeWindow string
	var errPrompt ErrorPrompt
	if menu.Configuration.Executable.CustomAutotypeWindowID == "" {
		activeWindow = robotgo.GetTitle()
	} else {
		command := []string{"sh", "-c", menu.Configuration.Executable.CustomAutotypeWindowID}
		activeWindow, errPrompt = executePrompt(command, nil)
	}

	if errPrompt.Error != nil || errPrompt.Cancelled {
		return &Entry{}, "", errPrompt
	}

	type pair struct {
		reg string
		ent Entry
		seq string
	}
	matches := make([]pair, 0)
	unmatches := make([]pair, 0, len(menu.Database.Entries))
	for _, e := range menu.Database.Entries {
		defaultSequence := "{USERNAME}{TAB}{PASSWORD}{ENTER}"
		if e.FullEntry.AutoType.DefaultSequence != "" {
			defaultSequence = e.FullEntry.AutoType.DefaultSequence
		}
		// [Regexp, KeySequence]
		mss := [][]string{}
		if e.FullEntry.AutoType.Associations != nil {
			for _, at := range e.FullEntry.AutoType.Associations {
				// Check if there's a window association, and if so, make sure it's a regexp
				if at.Window == "" {
					continue
				}
				ms := at.Window
				if !strings.HasPrefix(at.Window, "//") {
					// Replace all star globs with .*
					ms = strings.ReplaceAll(ms, "*", ".*")
				} else {
					// For regexp, remove the wrapping
					if len(ms) > 2 {
						ms = ms[2:]
					}
					if len(ms) > 2 {
						ms = ms[:len(ms)-2]
					}
				}
				seq := defaultSequence
				if at.KeystrokeSequence != "" {
					seq = at.KeystrokeSequence
				}
				mss = append(mss, []string{ms, seq})
			}
		}
		mss = append(mss, []string{".*" + e.FullEntry.GetContent("Title") + ".*", defaultSequence})

		for _, ms := range mss {
			reg, err := regexp.Compile(ms[0])
			if err != nil {
				continue
			}
			if reg.Match([]byte(activeWindow)) {
				matches = append(matches, pair{ms[0], e, ms[1]})
			} else {
				unmatches = append(unmatches, pair{ms[0], e, ms[1]})
			}
		}
	}

	var entry *Entry
	var keySeq string
	if len(matches) == 1 && !menu.Configuration.General.AutotypeConfirm {
		entry = &matches[0].ent
		keySeq = matches[0].seq

	} else {
		matches = append(matches, unmatches...)
		items := make([]string, len(matches))
		for i, m := range matches {
			items[i] = fmt.Sprintf("%-25s %-25s %-30s", m.ent.FullEntry.GetContent("Title"),
				m.ent.FullEntry.GetContent("UserName"), m.seq)
		}
		sel, err := PromptChoose(menu, items)
		ep := ErrorPrompt{}
		if err != ep || sel == -1 {
			if err.KbCustom1 {
				entry = &matches[sel].ent
				keySeq = "{USERNAME}"
			} else if err.KbCustom2 {
				entry = &matches[sel].ent
				keySeq = "{PASSWORD}"
			} else if err.KbCustom3 {
				// keySeq = "{USERNAME}{TAB}{PASSWORD}{ENTER}"
				keySeq = "{TOTP}"
				entry = &matches[sel].ent
			} else if err.KbCustom4 {
				keySeq = "{PASSWORD}{ENTER}"
				entry = &matches[sel].ent
			} else if err.KbCustom5 {
				keySeq = "{URL}"
				entry = &matches[sel].ent
			}

			return entry, keySeq, ep
		}
		entry = &matches[sel].ent
		keySeq = matches[sel].seq
	}
	if menu.Configuration.General.AutotypeSequence != "" {
		//  --autotypeSequence={USERNAME}
		keySeq = menu.Configuration.General.AutotypeSequence
	}
	return entry, keySeq, errPrompt
}

func getContent(e gokeepasslib.Entry, k string) string {
	k = strings.ToLower(k)
	for _, v := range e.Values {
		if strings.ToLower(v.Key) == k {
			return v.Value.Content
		}
	}
	return ""
}
