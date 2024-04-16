//go:build pprof
// +build pprof

package fzf

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/junegunn/fzf/src/util"
)

func (o *Options) initProfiling() error {
	if o.CPUProfile != "" {
		f, err := os.Create(o.CPUProfile)
		if err != nil {
			return fmt.Errorf("could not create CPU profile: %w", err)
		}

		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("could not start CPU profile: %w", err)
		}

		util.AtExit(func() {
			pprof.StopCPUProfile()
			if err := f.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "Error: closing cpu profile:", err)
			}
		})
	}

	stopProfile := func(name string, f *os.File) {
		if err := pprof.Lookup(name).WriteTo(f, 0); err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not write %s profile: %v\n", name, err)
		}
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: closing %s profile: %v\n", name, err)
		}
	}

	if o.MEMProfile != "" {
		f, err := os.Create(o.MEMProfile)
		if err != nil {
			return fmt.Errorf("could not create MEM profile: %w", err)
		}
		util.AtExit(func() {
			runtime.GC()
			stopProfile("allocs", f)
		})
	}

	if o.BlockProfile != "" {
		runtime.SetBlockProfileRate(1)
		f, err := os.Create(o.BlockProfile)
		if err != nil {
			return fmt.Errorf("could not create BLOCK profile: %w", err)
		}
		util.AtExit(func() { stopProfile("block", f) })
	}

	if o.MutexProfile != "" {
		runtime.SetMutexProfileFraction(1)
		f, err := os.Create(o.MutexProfile)
		if err != nil {
			return fmt.Errorf("could not create MUTEX profile: %w", err)
		}
		util.AtExit(func() { stopProfile("mutex", f) })
	}

	return nil
}
