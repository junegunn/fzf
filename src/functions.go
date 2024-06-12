package fzf

import (
	"os"
	"strings"
	"unsafe"
)

func WriteTemporaryFile(data []string, printSep string) string {
	f, err := os.CreateTemp("", "fzf-temp-*")
	if err != nil {
		// Unable to create temporary file
		// FIXME: Should we terminate the program?
		return ""
	}
	defer f.Close()

	f.WriteString(strings.Join(data, printSep))
	f.WriteString(printSep)
	return f.Name()
}

func removeFiles(files []string) {
	for _, filename := range files {
		os.Remove(filename)
	}
}

func stringBytes(data string) []byte {
	return unsafe.Slice(unsafe.StringData(data), len(data))
}

func byteString(data []byte) string {
	return unsafe.String(unsafe.SliceData(data), len(data))
}
