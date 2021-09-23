// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

const shpIgnoreFilename = ".shpignore"

// PackAndPush a local directory as-is into a container image. See
// remote.Option for optional options to the image push to the registry, for
// example to provide the appropriate access credentials.
func PackAndPush(ref name.Reference, directory string, options ...remote.Option) (name.Digest, error) {
	r, err := Pack(directory)
	if err != nil {
		return name.Digest{}, err
	}

	bundleLayer, err := tarball.LayerFromReader(r)
	if err != nil {
		return name.Digest{}, err
	}

	image, err := mutate.AppendLayers(empty.Image, bundleLayer)
	if err != nil {
		return name.Digest{}, err
	}

	image, err = mutate.Time(image, time.Unix(0, 0))
	if err != nil {
		return name.Digest{}, err
	}

	hash, err := image.Digest()
	if err != nil {
		return name.Digest{}, err
	}

	if err := remote.Write(ref, image, options...); err != nil {
		return name.Digest{}, err
	}

	return name.NewDigest(fmt.Sprintf("%s@%v",
		ref.Name(),
		hash.String(),
	))
}

// PullAndUnpack a container image layer content into a local directory. Analog
// to the bundle.PackAndPush function, optional remote.Option can be used to
// configure settings for the image pull, i.e. access credentials.
func PullAndUnpack(ref name.Reference, targetPath string, options ...remote.Option) (*containerreg.Image, error) {
	desc, err := remote.Get(ref, options...)
	if err != nil {
		return nil, err
	}

	image, err := desc.Image()
	if err != nil {
		return nil, err
	}

	rc := mutate.Extract(image)
	defer rc.Close()

	if err = Unpack(rc, targetPath); err != nil {
		return nil, err
	}

	return &image, nil
}

// Pack reads a directory and creates a tar stream with its content by:
// - storing all directories and regular files as-is,
// - dereferencing all symlinks and storing the respective target,
// - ignoring all files configured in .shpignore
func Pack(directory string) (io.Reader, error) {
	var split = func(path string) []string { return strings.Split(path, string(filepath.Separator)) }

	var write = func(w io.Writer, path string) error {
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		defer file.Close()

		_, err = io.Copy(w, file)
		return err
	}

	var followSymLink = func(path string) (string, os.FileInfo, error) {
		deref, err := os.Readlink(path)
		if err != nil {
			return "", nil, err
		}

		if !filepath.IsAbs(deref) {
			deref = filepath.Join(
				filepath.Dir(path),
				deref,
			)
		}

		info, err := os.Stat(deref)
		return deref, info, err
	}

	var patterns []gitignore.Pattern
	if file, err := os.Open(filepath.Join(directory, shpIgnoreFilename)); err == nil {
		defer file.Close()

		domain := split(directory)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if len(line) != 0 && !strings.HasPrefix(line, "#") {
				patterns = append(patterns, gitignore.ParsePattern(line, domain))
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	matcher := gitignore.NewMatcher(patterns)

	var buf bytes.Buffer
	var tw = tar.NewWriter(&buf)
	defer tw.Close()

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		// Bail out on path errors
		if err != nil {
			return err
		}

		// Skip files on the ignore list
		if matcher.Match(split(path), d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(directory, path)
		if err != nil {
			return err
		}

		switch {
		case info.Mode().IsDir():
			return tw.WriteHeader(header)

		case info.Mode().IsRegular():
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			return write(tw, path)

		case info.Mode()&os.ModeSymlink == os.ModeSymlink:
			deref, info, err := followSymLink(path)
			if err != nil {
				return err
			}

			header, err = tar.FileInfoHeader(info, deref)
			if err != nil {
				return err
			}

			header.Name, err = filepath.Rel(directory, path)
			if err != nil {
				return err
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			return write(tw, deref)

		default:
			return fmt.Errorf("unsupported file type: %s", path)
		}
	})

	return &buf, err
}

// Unpack reads a tar stream and writes the content into the local file system
// with all files and directories.
func Unpack(in io.Reader, targetPath string) error {
	var tr = tar.NewReader(in)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}

		var target = filepath.Join(targetPath, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}

		case tar.TypeReg:
			// Edge case in which that tarball did not have a directory entry
			dir, _ := filepath.Split(target)
			if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
				return err
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}

			file.Close()
			os.Chtimes(target, header.AccessTime, header.ModTime)

		default:
			return fmt.Errorf("provided tarball contains unsupported file type, only directories and regular files are supported")
		}
	}
}
