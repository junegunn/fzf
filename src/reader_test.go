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
	testRootPath, err = filepath.Abs(testRootPath)
	if err != nil {
		t.Errorf("error: %s", err)
		return
	}
	// when symlink is encountered and evaluated, the whole path is evaluated so
	// for comparison purposes we need to make sure test root is also evaluated
	// notable example is /tmp symlink to /private/tmp on macos
	testRootPath, err = filepath.EvalSymlinks(testRootPath)
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

	testWD := filepath.Join(testRootPath, "sup/wd")
	err = os.MkdirAll(testWD, 0777)
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
		"file0",
		"dir/file1",
		"dir/subdir/file2",
		"dir/subdir/another-file3",
		"sup/file4", // symlink2
		"sup/dir/file5",
		"sup/wd/file6",
		"sup/wd/dir/file7",
		"sup/wd/dir/target/subdir/file8", // symlink1
		"sup/wd/another-dir/subdir/file9",
		"sup/wd/another-dir/subdir/another-file10",
		"sup/wd/.dir/file11",
		"sup/wd/.file12",
		"target/file13", // symlink0
		"target/subdir/file14",
		"file15",                                    // symlink3
		"prefix/supdir/target/subdir/subdir/file16", // symlink4
		"prefix/supdir/target/subdir/subdir/file17",
		"prefix/supdir/target/subdir/another-subdir/file18",
		"prefix/supdir/file19",
		"prefix/target/dir/file20", // symlink5
		"prefix/target/dir/file21",
		"prefix/target2/file22", // symlink6
		"prefix/target2/file23",
		"prefix/file24", // symlink7
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
		// outward symlink
		"sup/wd/symlink0": "target",
		/*
			inward symlink
			note that the target dir can be reached via two paths, however it
			will be included in result set only once regardless the order
		*/
		"sup/wd/symlink1": "sup/wd/dir/target",
		/*
			upward symlink
			note that this symlink contains all of the working dir wd, but won't
			duplicate items because of the cycle
		*/
		"sup/wd/another-dir/subdir/symlink2": "sup",
		// file symlink
		"sup/wd/symlink3": "file15",
		// symlink chain
		"sup/wd/symlink4":               "prefix/supdir/target",
		"prefix/supdir/target/symlink5": "prefix/target",
		"prefix/target/symlink6":        "prefix/target2",
		"prefix/target2/symlink7":       "prefix/file24",
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
	// used as a set, values are optional aliases (all aliases are removed at once)
	expected := map[string]string{
		// files 0..3 excluded/unreachable
		"another-dir/subdir/symlink2/file4":     "",
		"another-dir/subdir/symlink2/dir/file5": "",
		"file6":                                 "",
		"dir/file7":                             "",
		"dir/target/subdir/file8":               "symlink1/subdir/file8",
		"symlink1/subdir/file8":                 "dir/target/subdir/file8",
		"another-dir/subdir/file9":              "",
		"another-dir/subdir/another-file10":     "",
		// file11 in hidden dir, so it gets skipped
		".file12":                               "",
		"symlink0/file13":                       "",
		"symlink0/subdir/file14":                "",
		"symlink3":                              "",
		"symlink4/subdir/subdir/file16":         "",
		"symlink4/subdir/subdir/file17":         "",
		"symlink4/subdir/another-subdir/file18": "",
		// file19 squeezed between two symlink trees and intentionally unreachable
		"symlink4/symlink5/dir/file20":        "",
		"symlink4/symlink5/dir/file21":        "",
		"symlink4/symlink5/symlink6/file22":   "",
		"symlink4/symlink5/symlink6/file23":   "",
		"symlink4/symlink5/symlink6/symlink7": "",
	}
	for _, s := range pushedStrings {
		s = filepath.ToSlash(s) // windows: normalize path separators for comparison purposes
		if alias, found := expected[s]; found {
			delete(expected, s)
			delete(expected, alias)
		} else {
			t.Errorf("unexpected file encountered: %s", s)
		}
	}
	for k := range expected {
		t.Errorf("didn't encounter expected file: %s", k)
	}
}
