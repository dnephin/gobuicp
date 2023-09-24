package main

import (
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

func readActionCacheFile(fh *os.File) (OutputID, error) {
	outid := OutputID{}
	if _, err := fh.Seek(0, 0); err != nil {
		return outid, err
	}

	// putIndexEntry(id ActionID, out OutputID, size int64, allowVerify bool) error {
	// entry := fmt.Sprintf("v1 %x %x %20d %20d\n", id, out, size, time.Now().UnixNano())
	aid := ActionID{}
	if _, err := fmt.Fscanf(fh, "v1 %x %x", &aid, &outid); err != nil {
		return outid, err
	}
	return outid, nil
}

// from src/cmd/internal/buildid/buildid.go

// hashToString converts the hash h to a string to be recorded
// in package archives and binaries as part of the build ID.
// We use the first 120 bits of the hash (5 chunks of 24 bits each) and encode
// it in base64, resulting in a 20-byte string. Because this is only used for
// detecting the need to rebuild installed files (not for lookups
// in the object file cache), 120 bits are sufficient to drive the
// probability of a false "do not need to rebuild" decision to effectively zero.
// We embed two different hashes in archives and four in binaries,
// so cutting to 20 bytes is a significant savings when build IDs are displayed.
// (20*4+3 = 83 bytes compared to 64*4+3 = 259 bytes for the
// more straightforward option of printing the entire h in base64).

// TODO: for fuzz testing stringToHash
func hashToString(h [32]byte) string {
	const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	const chunks = 5
	var dst [chunks * 4]byte
	for i := 0; i < chunks; i++ {
		v := uint32(h[3*i])<<16 | uint32(h[3*i+1])<<8 | uint32(h[3*i+2])
		dst[4*i+0] = b64[(v>>18)&0x3F]
		dst[4*i+1] = b64[(v>>12)&0x3F]
		dst[4*i+2] = b64[(v>>6)&0x3F]
		dst[4*i+3] = b64[v&0x3F]
	}
	return string(dst[:])
}
