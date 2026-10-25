package fzf

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/junegunn/fzf/src/util"
)

func TestReadFromCommand(t *testing.T) {
	strs := []string{}
	eb := util.NewEventBox()
	exec := util.NewExecutor("")
	reader := NewReader(
		func(s []byte) bool { strs = append(strs, string(s)); return true },
		eb, exec, false, true)

	reader.startEventPoller()

	// Check EventBox
	if eb.Peek(EvtReadNew) {
		t.Error("EvtReadNew should not be set yet")
	}

	// Normal command
	counter := 0
	ready := func() {
		counter++
	}
	reader.fin(reader.readFromCommand(`echo abc&&echo def`, nil, ready))
	if len(strs) != 2 || strs[0] != "abc" || strs[1] != "def" || counter != 1 {
		t.Errorf("%s", strs)
	}

	// Check EventBox again
	eb.WaitFor(EvtReadFin)

	// Wait should return immediately
	eb.Wait(func(events *util.Events) {
		events.Clear()
	})

	// EventBox is cleared
	if eb.Peek(EvtReadNew) {
		t.Error("EvtReadNew should not be set yet")
	}

	// Make sure that event poller is finished
	time.Sleep(readerPollIntervalMax)

	// Restart event poller
	reader.startEventPoller()

	// Failing command
	reader.fin(reader.readFromCommand(`no-such-command`, nil, ready))
	strs = []string{}
	if len(strs) > 0 || counter != 2 {
		t.Errorf("%s", strs)
	}

	// Check EventBox again
	if eb.Peek(EvtReadNew) {
		t.Error("Command failed. EvtReadNew should not be set")
	}
	if !eb.Peek(EvtReadFin) {
		t.Error("EvtReadFin should be set")
	}
}

func walkFiles(t *testing.T, roots ...string) []string {
	t.Helper()
	var mu sync.Mutex
	files := []string{}
	reader := NewReader(func(data []byte) bool {
		mu.Lock()
		defer mu.Unlock()
		files = append(files, filepath.FromSlash(string(data)))
		return true
	}, util.NewEventBox(), nil, false, false)
	if !reader.readFiles(roots, walkerOpts{file: true, follow: true}, nil) {
		t.Fatal("file walk failed")
	}
	slices.Sort(files)
	return files
}

func TestReadFilesSymlinkAcrossRoots(t *testing.T) {
	base := t.TempDir()
	app := filepath.Join(base, "app")
	shared := filepath.Join(base, "shared")
	frontend := filepath.Join(shared, "frontend")
	os.MkdirAll(app, 0755)
	os.MkdirAll(frontend, 0755)
	os.WriteFile(filepath.Join(shared, "docs.txt"), nil, 0644)
	os.WriteFile(filepath.Join(frontend, "index.js"), nil, 0644)
	if err := os.Symlink(filepath.Join("..", "shared"), filepath.Join(app, "shared")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	viaApp := []string{
		filepath.Join(app, "shared", "docs.txt"),
		filepath.Join(app, "shared", "frontend", "index.js"),
	}
	both := append(slices.Clone(viaApp), filepath.Join(frontend, "index.js"))
	slices.Sort(both)
	for _, tc := range []struct {
		roots []string
		want  []string
	}{
		{[]string{app}, viaApp},
		{[]string{app, frontend}, both},
		{[]string{frontend, app}, both},
	} {
		if files := walkFiles(t, tc.roots...); !slices.Equal(files, tc.want) {
			t.Errorf("roots = %v, files = %v, want %v", tc.roots, files, tc.want)
		}
	}
}

func TestReadFilesSkipsAncestorSymlinks(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "project")
	os.Mkdir(project, 0755)
	file := filepath.Join(project, "file.txt")
	os.WriteFile(file, nil, 0644)
	os.WriteFile(filepath.Join(base, "outside.txt"), nil, 0644)
	for name, target := range map[string]string{"parent": "..", "self": "."} {
		if err := os.Symlink(target, filepath.Join(project, name)); err != nil {
			t.Skipf("cannot create symlink: %v", err)
		}
	}
	if files := walkFiles(t, project); !slices.Equal(files, []string{file}) {
		t.Errorf("files = %v, want [%s]", files, file)
	}
}
