//go:build !pprof
// +build !pprof

package fzf

func (o *Options) initProfiling() error {
	if o.CPUProfile != "" || o.MEMProfile != "" || o.BlockProfile != "" || o.MutexProfile != "" {
		errorExit("error: profiling not supported: FZF must be built with '-tags=pprof' to enable profiling")
	}
	return nil
}
