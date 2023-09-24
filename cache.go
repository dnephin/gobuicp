package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// from cmd/go/internal/cache/cache.go

// HashSize is the number of bytes in a hash.
const HashSize = 32

// An ActionID is a cache action key, the hash of a complete description of a
// repeatable computation (command line, environment variables,
// input file contents, executable contents).
type ActionID [HashSize]byte

// An OutputID is a cache output key, the hash of an output of a computation.
type OutputID [HashSize]byte

// fileName returns the name of the file corresponding to the given id.
func fileName(id [HashSize]byte, key string) string {
	return filepath.Join(fmt.Sprintf("%02x", id[0]), fmt.Sprintf("%x", id)+"-"+key)
}

var errNoOutputID = errors.New("no outputID")

func readActionCacheFile(fh *os.File) (OutputID, error) {
	outid := [HashSize]byte{}
	if _, err := fh.Seek(0, 0); err != nil {
		return outid, err
	}

	// putIndexEntry(id ActionID, out OutputID, size int64, allowVerify bool) error {
	// entry := fmt.Sprintf("v1 %x %x %20d %20d\n", id, out, size, time.Now().UnixNano())
	var aid string
	t := outid[:0]
	n, err := fmt.Fscanf(fh, "v1 %v %x", &aid, &t)
	if err != nil {
		return outid, err
	}
	// TODO: can this happen?
	if n < 2 {
		return outid, errNoOutputID
	}
	// TODO: why isn't outid populated?
	return OutputID(t[:HashSize]), nil
}
