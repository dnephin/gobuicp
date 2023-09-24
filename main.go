package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := setupFlags(args)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := copyBuildCache(opts); err != nil {
		return err
	}
	return nil
}

type options struct {
	fromBaseDir         string
	toBaseDir           string
	actionGraphFilename string
}

func setupFlags(args []string) (options, error) {
	opt := options{}

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.StringVar(&opt.fromBaseDir, "from", "",
		"copy files from cache directories under this base directory")
	flags.StringVar(&opt.toBaseDir, "to", "",
		"copy files to cache directories under this base directory")
	flags.StringVar(&opt.actionGraphFilename, "actiongraph", "actiongraph.json",
		"filename of the -debug-actiongraph output for the build cache")

	if err := flags.Parse(args[1:]); err != nil {
		return opt, err
	}
	return opt, nil
}

func copyBuildCache(opts options) error {
	// TODO: different on macos
	fromDir := filepath.Join(opts.fromBaseDir, ".cache/go-build")
	toDir := filepath.Join(opts.toBaseDir, ".cache/go-build")
	os.MkdirAll(toDir, 0755)

	pkgs, err := parseActionGraph(opts.actionGraphFilename)
	if err != nil {
		return err
	}

	var count int
	var size int64
	for _, pkg := range pkgs {
		logger := slog.With("pkg", pkg.Package, "mode", pkg.Mode)

		if pkg.ActionID == "" {
			logger.Warn("no action ID")
			continue
		}
		filename, err := findActionCacheFile(fromDir, pkg.ActionID)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			logger.Warn("no file matched", "prefix", filename)
			continue
		case err != nil:
			return err
		}

		srcFH, n, err := copyFile(filepath.Join(toDir, filename), filepath.Join(fromDir, filename))
		if err != nil {
			return err
		}
		count++
		size += n

		outputID, err := readActionCacheFile(srcFH)
		srcFH.Close() // close after read
		switch {
		case errors.Is(err, errNoOutputID):
			logger.Warn("no output file")
			continue
		case err != nil:
			return err
		}

		filename = fileName(outputID, "d")
		srcFH, n, err = copyFile(filepath.Join(toDir, filename), filepath.Join(fromDir, filename))
		if err != nil {
			return err
		}
		srcFH.Close()
		count++
		size += n
	}
	// TODO: humanize bytes
	slog.Info("Copied go build cache", "files", count, "bytes", size)
	return nil
}

func findActionCacheFile(cacheDir string, actionID string) (string, error) {
	prefix, err := base64.URLEncoding.DecodeString(actionID)
	if err != nil {
		return "", err
	}

	subdir := fmt.Sprintf("%02x", prefix[0])
	dir := filepath.Join(cacheDir, subdir)
	file, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	defer file.Close()
	names, err := file.Readdirnames(0)
	if err != nil {
		return "", err
	}

	filePrefix := fmt.Sprintf("%x", prefix)
	for _, name := range names {
		// TODO: check -a suffix as well?
		if strings.HasPrefix(name, filePrefix) {
			return filepath.Join(subdir, name), nil
		}
	}
	return filePrefix, fs.ErrNotExist
}

type actionGraphPkg struct {
	ActionID  string
	Package   string
	Mode      string
	NeedBuild bool
}

func parseActionGraph(filename string) ([]actionGraphPkg, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close()

	var target []actionGraphPkg
	err = json.NewDecoder(fh).Decode(&target)
	return target, err
}

func copyFile(dest, src string) (*os.File, int64, error) {
	srcFH, err := os.Open(src)
	if err != nil {
		return nil, 0, err
	}

	os.MkdirAll(filepath.Dir(dest), 0755)
	destFH, err := os.Create(dest)
	if err != nil {
		srcFH.Close()
		return nil, 0, err
	}
	size, err := io.Copy(destFH, srcFH)
	if err != nil {
		srcFH.Close()
		destFH.Close()
		return nil, 0, err
	}
	if err := destFH.Close(); err != nil {
		srcFH.Close()
		return nil, 0, err
	}

	slog.Debug("copied file", "size", size, "from", srcFH.Name(), "to", destFH.Name())
	return srcFH, size, nil
}
