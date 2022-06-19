package format

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/zeromicro/go-zero/core/errorx"
	"github.com/zeromicro/zero-api/format"
)

var (
	// VarBoolUseStdin describes whether to use stdin or not.
	VarBoolUseStdin bool
	// VarBoolSkipCheckDeclare describes whether to skip.
	VarBoolSkipCheckDeclare bool
	// VarStringDir describes the directory.
	VarStringDir string
	// VarBoolIgnore describes whether to ignore.
	VarBoolIgnore bool
)

// GoFormatApi format api file
func GoFormatApi(_ *cobra.Command, _ []string) error {
	var be errorx.BatchError
	if VarBoolUseStdin {
		if err := processFile("<standard input>", nil, os.Stdin); err != nil {
			be.Add(err)
		}
	} else {
		if len(VarStringDir) == 0 {
			return errors.New("missing -dir")
		}

		_, err := os.Lstat(VarStringDir)
		if err != nil {
			return errors.New(VarStringDir + ": No such file or directory")
		}

		err = filepath.Walk(VarStringDir, func(path string, fi os.FileInfo, errBack error) (err error) {
			ext := filepath.Ext(path)
			if ext != ".api" {
				return nil
			}

			if err := processFile(path, fi, nil); err != nil {
				be.Add(err)
			}
			return nil
		})
		be.Add(err)
	}

	if be.NotNil() {
		fmt.Fprint(os.Stderr, be.Err())
		os.Exit(1)
	}

	return be.Err()
}

func ApiFormatByPath(fileName string) error {
	return processFile(fileName, nil, nil)
}

func processFile(fileName string, info fs.FileInfo, in io.Reader) error {
	if in == nil {
		var err error
		in, err = os.Open(fileName)
		if err != nil {
			return err
		}
	}

	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	res, err := format.Source(src, fileName)
	if err != nil {
		return err
	}

	// write to file
	perm := info.Mode().Perm()
	backName, err := backupFile(fileName+".", src, perm)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, res, perm)
	if err != nil {
		_ = os.Rename(backName, fileName)
		return err
	}

	err = os.Remove(backName)
	return err
}

const chmodSupported = runtime.GOOS != "windows"

func backupFile(filename string, data []byte, perm fs.FileMode) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(filename), filepath.Base(filename))
	if err != nil {
		return "", err
	}
	backname := f.Name()
	if chmodSupported {
		err := f.Chmod(perm)
		if err != nil {
			_ = f.Close()
			_ = os.Remove(backname)
			return backname, err
		}
	}

	_, err = f.Write(data)
	if err1 := f.Close(); err1 != nil {
		err = err1
	}
	return backname, err
}
