//go:build !pprof
// +build !pprof

package fzf

import "errors"

func (o *Options) initProfiling() error {
	if o.CPUProfile != "" || o.MEMProfile != "" || o.BlockProfile != "" || o.MutexProfile != "" {
		return errors.New("error: profiling not supported: FZF must be built with '-tags=pprof' to enable profiling")
	}
	return nil
}
