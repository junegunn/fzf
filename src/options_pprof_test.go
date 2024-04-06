//go:build pprof
// +build pprof

package fzf

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/junegunn/fzf/src/util"
)

// runInitProfileTests is an internal flag used TestInitProfiling
var runInitProfileTests = flag.Bool("test-init-profile", false, "run init profile tests")

func TestInitProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("short test")
	}

	// Run this test in a separate process since it interferes with
	// profiling and modifies the global atexit state. Without this
	// running `go test -bench . -cpuprofile cpu.out` will fail.
	if !*runInitProfileTests {
		t.Parallel()

		// Make sure we are not the child process.
		if os.Getenv("_FZF_CHILD_PROC") != "" {
			t.Fatal("already running as child process!")
		}

		cmd := exec.Command(os.Args[0],
			"-test.timeout", "30s",
			"-test.run", "^"+t.Name()+"$",
			"-test-init-profile",
		)
		cmd.Env = append(os.Environ(), "_FZF_CHILD_PROC=1")

		out, err := cmd.CombinedOutput()
		out = bytes.TrimSpace(out)
		if err != nil {
			t.Fatalf("Child test process failed: %v:\n%s", err, out)
		}
		// Make sure the test actually ran
		if bytes.Contains(out, []byte("no tests to run")) {
			t.Fatalf("Failed to run test %q:\n%s", t.Name(), out)
		}
		return
	}

	// Child process

	tempdir := t.TempDir()
	t.Cleanup(util.RunAtExitFuncs)

	o := Options{
		CPUProfile:   filepath.Join(tempdir, "cpu.prof"),
		MEMProfile:   filepath.Join(tempdir, "mem.prof"),
		BlockProfile: filepath.Join(tempdir, "block.prof"),
		MutexProfile: filepath.Join(tempdir, "mutex.prof"),
	}
	if err := o.initProfiling(); err != nil {
		t.Fatal(err)
	}

	profiles := []string{
		o.CPUProfile,
		o.MEMProfile,
		o.BlockProfile,
		o.MutexProfile,
	}
	for _, name := range profiles {
		if _, err := os.Stat(name); err != nil {
			t.Errorf("Failed to create profile %s: %v", filepath.Base(name), err)
		}
	}

	util.RunAtExitFuncs()

	for _, name := range profiles {
		if _, err := os.Stat(name); err != nil {
			t.Errorf("Failed to write profile %s: %v", filepath.Base(name), err)
		}
	}
}
