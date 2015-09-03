//Package filesystem provides filesystem based Stages for Slurp.
package fs

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-gonzo/fs/glob"
	"github.com/omeid/gonzo/context"
	"github.com/omeid/gonzo"
)

var ErrIsDir = errors.New("path is a directory.")

// A simple helper function that opens the file from the given path and
// returns a pointer to a gonzo.File or an error.
func Read(path string) (gonzo.File, error) {
	Stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if Stat.IsDir() {
		return nil, ErrIsDir
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return gonzo.NewFile(f, gonzo.FileInfoFrom(Stat)), nil
}

//Src returns a channel of gonzo.Files that match the provided patterns.
//TODO: ADD support for prefix to avoid all the util.Trims
func Src(ctx context.Context, globs ...string) gonzo.Pipe {

	files := make(chan gonzo.File)
	pipe := gonzo.NewPipe(ctx, files)

	//TODO: Parse globs here, check for invalid globs, split them into "filters".
	go func() {

		var err error
		defer close(files)

		fileslist, err := glob.Glob(globs...)
		if err != nil {
			ctx.Error(err)
			return
		}

		for mp := range fileslist {

			var (
				file gonzo.File
				base = glob.Dir(mp.Glob)
				name = mp.Name
			)

			file, err = Read(mp.Name)
			ctx = context.WithValue(ctx, "file", name)
			if err != nil {
				ctx.Error(err)
				return
			}

			file.FileInfo().SetBase(base)
			file.FileInfo().SetName(name)
			files <- file
		}

	}()

	return pipe
}

// Dest writes the files from the input channel to the dst folder and closes the files.
// It never returns Files.
func Dest(dst string) gonzo.Stage {
	return func(ctx context.Context, files <-chan gonzo.File, out chan<- gonzo.File) error {

		for {
			select {
			case file, ok := <-files:
				if !ok {
					return nil
				}

				name := file.FileInfo().Name()
				path := filepath.Join(dst, filepath.Dir(name))
				err := os.MkdirAll(path, 0700)
				if err != nil {
					return err
				}

				if file.FileInfo().IsDir() {
					out <- file
					continue
				}

				content, err := ioutil.ReadAll(file)
				if err != nil {
					file.Close()
					return err
				}

				ctx = context.WithValue(ctx, "path", path)
				ctx.Infof("Writing %s", name)
				err = writeFile(filepath.Join(dst, name), content)
				if err != nil {
					return err
				}

				out <- gonzo.NewFile(ioutil.NopCloser(bytes.NewReader(content)), file.FileInfo())

			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func writeFile(path string, content []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(content)
	return err
}
