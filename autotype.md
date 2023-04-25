Autotype
========

This patch adds support for performing autotype in an external utility, reference tickets #3 and #8

I've been using this for the past week or so, fixing behaviors and bugs and improving the autotype program; the main changes have been stable for about a week, and I added the main sequence parsing code a couple of days ago and have updated that code today. The sequence parsing code is shared with `quasiauto` through copy/paste.

At a high level, this consists of four changes:

1. Switches to control the behavior (`clientserver.go`, `config.go`, `kpmenulib.go`)
2. A command for getting the currently focused window ID, and matching it against entries in the database (`prompt.go`)
3. A command for executing the autotype (also `prompt.go`)
4. The ability to parse entry key sequences, necessary for choosing which values to send to the autotyper (`sequence.go`, `sequence_test.go`)

The new flags are:

- A switch for disabling autotype on the server process
- A switch to force user confirmation when the server gets an autotype request
- A switch to disable auto-detection of the active window (always show entry selection dialog)
- Switches to set the external programs for getting the active window title, and for executing the autotype
- A flag to trigger an autotype, sent by the client to the server

Autotyping is handled externally, which avoids bringing in more code dependencies but introduces external dependencies. By default, the window ID program is [xdotool](https://www.semicomplete.com/projects/xdotool/), and the autotype program is [quasiauto](https://hg.sr.ht/~ser/quasiauto). The former is a common tool and will be available in (probably) any Linux distribution package manager; the latter can be installed either by downloading a pre-compiled executable, or compiling it with Go.

When triggered by the client, the server launches the `customAutotypeWindowID` program (default: `xdotool`) to identify the currently active window. It then scans the database for a matching entry, using `AutoType.Association` or the `Title` if no association is set. If it finds a matching entry, it launches `customAutotypeTyper` (default: `quasiauto`) and writes the key sequence and the requested fields to the process' STDIN.

To get access to the key sequences, https://github.com/tobischo/gokeepasslib/issues/68 needed to be fixed. @tobischo pushed the fixes, and this patch consequently updates the version of gokeepasslib.

`prompt.go` has a controversial refactoring. Much of the code to generate the Exec() command (~44 LOC) was duplicated almost identically across 5 functions. I added two new `Prompt...()` functions, which would have duplicated this code even more, so I factored that code out into a helper function `getCommand()`. This changed code that would not necessarily change just for this patch.

Limitations
-----------

Only Linux is currently supported. This is due to the dependency on `xdotool`, which is necessitated by [a bug in `robotgo`](https://github.com/go-vgo/robotgo/issues/258) that prevents using that library to get window titles. While it is possible that external tools for Darwin and Windows exist that perform the same function as `xdotool`, I have access to neither systems and so can't test it. The current code does not prevent such a solution, and those tools could be configured with `--customAutotypeWindowID`.  If or when the `robotgo` bug is fixed, I can add window title ID to `quasiauto`, and it should be a cross-platform solution and reduce the external dependencies.

Testing
-------

The most basic test is to create a dummy database with an entry containing an autotype window match of `Window*`. Run `kpmenu` using `/usr/bin/printf` and `/usr/bin/xsel` to match the entry and dump the output. For example, assuming you've created a `testdata.kdbx` with the entry:

```
./kpmenu -d ./testdata.kdbx --customAutotypeWindowID '/usr/bin/printf "%s\040%s\n" Window Title' --customAutotypeTyper '/usr/bin/xsel -b'
```

and in a second terminal:

```
./kpmenu -d ./testdata.kdbx  --autotype
```

Then running `xsel -b` in the second terminal should print out the username and password (and OTP, if configured) of the entry.

**Note the funny printf** is because the code splits the custom command arguments on spaces, so no single argument can contain spaces. \040 is the ASCII control sequence for *space*.

A more full test uses [`zenity`](https://gitlab.gnome.org/GNOME/zenity) (the GUI dialog command-line runner) and has `quasiauto` in the path:

Shell 1:
```
./kpmenu -d ./testdata.kdbx
```
Shell 2:
```
zenity --password --username --title "Window Title"
```
Shell 3 (be prepared to switch back to the `zenity` window):
```
sleep 2; ./kpmenu -d ./testdata.kdbx --autotype
sleep 2; ./kpmenu -d ./testdata.kdbx --autotype
```
and watch with delight and amazement as the dialog is filled in. If you add OTP to the entry, a `zenity` dialog that demonstrates TOTP is:
```
zenity --forms --add-entry "Username" --add-password "Password" --add-entry "TOTP" --title "Window Title Here"
```
Of course, the design intends the `--autotype` call to be bound to a window manager hot-key.

Future
------

The code works, and is an minimum-viable solution given the limitations imposed by dependency bugs. It's useful as is, so I'm submitting a PR.

Despite my design desires, it is turning out to be more challenging to implement a user-controlled quasitype, so the first version implements only autotype. I have no doubt I can resolve the issues, and it should be easy to do a mouse-event driven process; these -- like the sequence bug work-around -- can evolve by changing `quasiauto` and will not require changes to `kpmenu`.

`sequence.go` will probably change, s.t. instead of hard-coding the KeePass values it'll be parsed out of a CVS -- switches will become loops. It'll shorten the code significantly at the cost of speed, but it's not in a critical path section so it should be fine.
