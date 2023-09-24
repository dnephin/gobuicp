package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	for _, pkg := range pkgs {
		aid, err := base64.URLEncoding.DecodeString(pkg.ActionID)
		if err != nil {
			return err
		}
		fmt.Println(aid, len(aid))

		filename := fileName(ActionID(aid), "a")
		fmt.Println(filename)
		srcFH, err := copyFile(filepath.Join(toDir, filename), filepath.Join(fromDir, filename))
		if err != nil {
			return err
		}

		outputID, err := readActionCacheFile(srcFH)
		srcFH.Close() // close after read
		if err != nil {
			return err
		}
		filename = fileName(outputID, "d")

		srcFH, err = copyFile(filepath.Join(toDir, filename), filepath.Join(fromDir, filename))
		if err != nil {
			return err
		}
		srcFH.Close()
	}
	return nil
}

type actionGraphPkg struct {
	ActionID  string
	Package   string
	Mode      string
	NeedBuild string
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

func copyFile(dest, src string) (*os.File, error) {
	srcFH, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	destFH, err := os.Create(dest)
	if err != nil {
		srcFH.Close()
		return nil, err
	}
	_, err = io.Copy(destFH, srcFH)
	if err != nil {
		srcFH.Close()
		destFH.Close()
		return nil, err
	}
	if err := destFH.Close(); err != nil {
		srcFH.Close()
		return nil, err
	}
	return srcFH, nil
}
