package fzf

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/junegunn/fzf/src/util"
)

func TestReadFromCommand(t *testing.T) {
	strs := []string{}
	eb := util.NewEventBox()
	reader := NewReader(
		func(s []byte) bool { strs = append(strs, string(s)); return true },
		eb, false, true)

	reader.startEventPoller()

	// Check EventBox
	if eb.Peek(EvtReadNew) {
		t.Error("EvtReadNew should not be set yet")
	}

	// Normal command
	reader.fin(reader.readFromCommand(nil, `echo abc&&echo def`))
	if len(strs) != 2 || strs[0] != "abc" || strs[1] != "def" {
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
	reader.fin(reader.readFromCommand(nil, `no-such-command`))
	strs = []string{}
	if len(strs) > 0 {
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

// create temporary file structure, run fzf on it and check the files it saw
func TestReadFiles(t *testing.T) {
	pushedStrings := []string{}
	pusher := func(s []byte) bool {
		pushedStrings = append(pushedStrings, string(s))
		return true
	}
	reader := NewReader(pusher, util.NewEventBox(), false, true)

	// setup test dir
	testRootPath, err := ioutil.TempDir("", "fzf-test-walk-")
	if err != nil {
		t.Errorf("error: %s", err)
		return
	}
	defer os.RemoveAll(testRootPath)

	// create and change to a fzf's working dir
	originalWD, err := os.Getwd()
	if err != nil {
		t.Errorf("error: %s", err)
		return
	}
	defer os.Chdir(originalWD)

	testWD := filepath.Join(testRootPath, "wd")
	err = os.Mkdir(testWD, 0777)
	if err != nil {
		t.Errorf("error: %s", err)
		return
	}
	err = os.Chdir(testWD)
	if err != nil {
		t.Errorf("error: %s", err)
		return
	}

	// create test files
	files := []string{
		"excludedFile",
		"excludedDir/foo",
		"wd/includedFile",
		"wd/includedDir/foo",
		"wd/includedDir/bar",
		"symlinkTarget/foo",
	}
	for _, relFilePath := range files {
		absFilePath := filepath.Join(testRootPath, relFilePath)
		absDirPath, _ := filepath.Split(absFilePath)

		// create all required dirs
		err = os.MkdirAll(absDirPath, 0777)
		if err != nil {
			t.Errorf("error: %s", err)
		}

		// touch the file
		absFile, err := os.Create(absFilePath)
		if err != nil {
			t.Errorf("error: %s", err)
		}
		absFile.Close()
	}

	// create test symlinks
	symlinks := map[string]string{
		"wd/includedSymlink": "symlinkTarget",
	}
	for relSymPath, relFilePath := range symlinks {
		absSymPath := filepath.Join(testRootPath, relSymPath)
		absFilePath := filepath.Join(testRootPath, relFilePath)
		absDirPath, _ := filepath.Split(absSymPath)

		// create all required dirs
		err = os.MkdirAll(absDirPath, 0777)
		if err != nil {
			t.Errorf("error: %s", err)
		}

		// create symlink
		err := os.Symlink(absFilePath, absSymPath)
		if err != nil {
			switch e := err.(type) {
			case *os.LinkError:
				if util.IsWindows() && e.Err.Error() == "A required privilege is not held by the client." {
					t.Skip("Skipped: this test requires admin privileges to create symlinks.")
				}
				t.Errorf("error: %s", e)
			default:
				t.Errorf("error: %s", e)
			}
		}
	}

	// make fzf read the files
	ok := reader.readFiles()
	if !ok {
		t.Error("error: readFiles() indicated error")
	}

	// check the read files
	expected := map[string]interface{}{ // used as a set, ignore values
		`includedFile`:    nil,
		`includedDir/foo`: nil,
		`includedDir/bar`: nil,
		`includedSymlink`: nil, // symlink is not followed
	}
	for _, s := range pushedStrings {
		s = filepath.ToSlash(s) // windows: normalize path separators for comparison purposes
		if _, found := expected[s]; found {
			delete(expected, s)
		} else {
			t.Errorf("unexpected file encountered: %s", s)
		}
	}
	for k := range expected {
		t.Errorf("didn't encounter expected file: %s", k)
	}
}
