//go:build windows

package util

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestNewWindowsExecutor(t *testing.T) {
	for _, test := range []struct {
		name      string
		envShell  string
		withShell string
		shell     string
		shellType shellType
		args      []string
	}{
		{"default", "", "", "cmd", shellTypeCmd, []string{"/s/c"}},
		{"cmd_env", "cmd.exe", "", "cmd.exe", shellTypeCmd, []string{"/s/c"}},
		{"cmd_path_env", `C:\Windows\System32\cmd.exe`, "", `C:\Windows\System32\cmd.exe`, shellTypeCmd, []string{"/s/c"}},
		{"powershell_env", "powershell.exe", "", "powershell.exe", shellTypePowerShell, []string{"-NoProfile", "-Command"}},
		{"pwsh_env", "pwsh.exe", "", "pwsh.exe", shellTypePwsh, []string{"-NoProfile", "-Command"}},
		{"custom_env", "custom-shell", "", "custom-shell", shellTypeUnknown, []string{"-c"}},
		{"cmd_explicit", "powershell.exe", "cmd.exe /d /c", "cmd.exe", shellTypeCmd, []string{"/d", "/c"}},
		{"cmd_path_explicit", "", `C:\Windows\System32\cmd.exe /d /s /c`, `C:\Windows\System32\cmd.exe`, shellTypeCmd, []string{"/d", "/s", "/c"}},
		{"powershell_explicit", "cmd.exe", "powershell.exe -NoLogo -NonInteractive -Command", "powershell.exe", shellTypePowerShell, []string{"-NoLogo", "-NonInteractive", "-Command"}},
		{"pwsh_explicit", "cmd.exe", "pwsh.exe -NoLogo -Command", "pwsh.exe", shellTypePwsh, []string{"-NoLogo", "-Command"}},
		{"custom_explicit", "cmd.exe", "custom-shell --custom -e", "custom-shell", shellTypeUnknown, []string{"--custom", "-e"}},
		{"cmd_no_flags", "", "cmd.exe", "cmd.exe", shellTypeCmd, []string{}},
		{"powershell_no_flags", "", "powershell.exe", "powershell.exe", shellTypePowerShell, []string{}},
		{"pwsh_no_flags", "", "pwsh.exe", "pwsh.exe", shellTypePwsh, []string{}},
		{"custom_no_flags", "", "custom-shell", "custom-shell", shellTypeUnknown, []string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SHELL", test.envShell)
			x := NewExecutor(test.withShell)
			if x.shell != test.shell || x.shellType != test.shellType || !reflect.DeepEqual(x.args, test.args) {
				t.Errorf("got shell=%q type=%v args=%q; want shell=%q type=%v args=%q",
					x.shell, x.shellType, x.args, test.shell, test.shellType, test.args)
			}
		})
	}
}

func TestWindowsExecutorQuoteEntry(t *testing.T) {
	entry := `hello 한글 O'Brien & "quoted"`
	for _, test := range []struct {
		withShell string
		want      string
	}{
		{"cmd.exe /d /c", `^"hello 한글 O'Brien ^& \^"quoted\^"^"`},
		{"powershell.exe -NoProfile -Command", `'hello 한글 O''Brien & \"quoted\"'`},
		{"pwsh.exe -NoProfile -Command", `'hello 한글 O''Brien & \"quoted\"'`},
		{"custom-shell -c", `'hello 한글 O'\''Brien & "quoted"'`},
	} {
		t.Run(test.withShell, func(t *testing.T) {
			if got := NewExecutor(test.withShell).QuoteEntry(entry); got != test.want {
				t.Errorf("got %q; want %q", got, test.want)
			}
		})
	}
}

func TestWindowsExecutorProcessGroup(t *testing.T) {
	for _, test := range []struct {
		withShell string
		flags     uint32
		args      []string
	}{
		{"cmd.exe /d /c", 0, nil},
		{"powershell.exe -NoLogo -Command", 0, []string{"-NoLogo", "-Command"}},
		{"pwsh.exe -NoLogo -Command", windows.CREATE_NEW_PROCESS_GROUP, []string{"-NonInteractive", "-NoLogo", "-Command"}},
		{"custom-shell -e", windows.CREATE_NEW_PROCESS_GROUP, []string{"-e"}},
	} {
		t.Run(test.withShell, func(t *testing.T) {
			x := NewExecutor(test.withShell)
			originalArgs := append([]string{}, x.args...)
			for _, setpgid := range []bool{false, true} {
				cmd := x.ExecCommand("echo hello", setpgid)
				var wantFlags uint32
				if setpgid {
					wantFlags = test.flags
				}
				if cmd.SysProcAttr.CreationFlags != wantFlags {
					t.Errorf("setpgid=%v: got flags=%v; want %v", setpgid, cmd.SysProcAttr.CreationFlags, wantFlags)
				}
				if strings.HasPrefix(test.withShell, "cmd") {
					if cmd.SysProcAttr.CmdLine != `/d /c "echo hello"` {
						t.Errorf("got CMD command line %q", cmd.SysProcAttr.CmdLine)
					}
				} else {
					wantArgs := test.args
					if !setpgid && strings.HasPrefix(test.withShell, "pwsh") {
						wantArgs = wantArgs[1:]
					}
					want := append(append([]string{cmd.Args[0]}, wantArgs...), "echo hello")
					if !reflect.DeepEqual(cmd.Args, want) {
						t.Errorf("setpgid=%v: got args=%q; want %q", setpgid, cmd.Args, want)
					}
				}
			}
			if !reflect.DeepEqual(x.args, originalArgs) {
				t.Errorf("ExecCommand changed executor arguments: %q", x.args)
			}
		})
	}
}

func TestWindowsExecutorQuotedCommand(t *testing.T) {
	// Run the test binary as a helper so this regression needs no external tools.
	// Both its path and its argument exercise Windows shell quoting.
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "quoted path 한글 & O'Brien")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(dir, "shell-probe.exe")
	if err := os.WriteFile(helper, data, 0700); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		shell     string
		withShell string
		prefix    string
	}{
		{"cmd.exe", "", ""},
		{"cmd.exe", "cmd.exe /d /c", ""},
		{"powershell.exe", "", "& "},
		{"powershell.exe", "powershell.exe -NoProfile -NonInteractive -Command", "& "},
		{"pwsh.exe", "", "& "},
		{"pwsh.exe", "pwsh.exe -NoProfile -Command", "& "},
	} {
		name := "default"
		if test.withShell != "" {
			name = "explicit"
		}
		t.Run(test.shell+"/"+name, func(t *testing.T) {
			if _, err := exec.LookPath(test.shell); err != nil {
				t.Skipf("%s is unavailable: %v", test.shell, err)
			}
			t.Setenv("SHELL", test.shell)
			x := NewExecutor(test.withShell)
			entry := `hello 한글 O'Brien &|<>()^%!`
			if test.prefix == "" {
				entry += ` "quoted" C:\trailing\`
			}
			command := test.prefix + `"` + helper + `" ` + x.QuoteEntry("-test.run=TestWindowsExecutorHelper") + " -- " + x.QuoteEntry(entry)
			cmd := x.ExecCommand(command, true)
			cmd.Env = append(os.Environ(), "FZF_WINDOWS_EXECUTOR_HELPER=1")
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			cmd.WaitDelay = 10 * time.Second
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			timer := time.AfterFunc(10*time.Second, func() { _ = cmd.Process.Kill() })
			defer timer.Stop()
			if err := cmd.Wait(); err != nil {
				t.Fatalf("command failed: %v\n%s", err, output.String())
			}
			var got []string
			if err := json.Unmarshal(output.Bytes(), &got); err != nil {
				t.Fatalf("invalid helper output: %v\n%s", err, output.String())
			}
			if !reflect.DeepEqual(got, []string{entry}) {
				t.Errorf("got arguments %q; want %q", got, []string{entry})
			}
		})
	}
}

func TestWindowsExecutorHelper(t *testing.T) {
	if os.Getenv("FZF_WINDOWS_EXECUTOR_HELPER") != "1" {
		return
	}
	for index, arg := range os.Args {
		if arg == "--" {
			if err := json.NewEncoder(os.Stdout).Encode(os.Args[index+1:]); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	os.Exit(1)
}
