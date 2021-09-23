// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

// The algorithm to mutate the image was inspired by
// https://github.com/google/go-containerregistry/blob/main/cmd/crane/cmd/mutate.go

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	containerreg "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/spf13/pflag"
)

// ExitError is an error which has an exit code to be used in os.Exit() to
// return both an exit code and an error message
type ExitError struct {
	Code    int
	Message string
	Cause   error
}

func (e ExitError) Error() string {
	return fmt.Sprintf("%s (exit code %d)", e.Message, e.Code)
}

// headerTransport sets headers on outgoing requests
type headerTransport struct {
	httpHeaders map[string]string
	inner       http.RoundTripper
}

// RoundTrip implements http.RoundTripper
func (ht *headerTransport) RoundTrip(in *http.Request) (*http.Response, error) {
	for k, v := range ht.httpHeaders {
		if http.CanonicalHeaderKey(k) == "User-Agent" {
			// Docker sets this, which is annoying, since we're not docker.
			// We might want to revisit completely ignoring this.
			continue
		}
		in.Header.Set(k, v)
	}

	return ht.inner.RoundTrip(in)
}

type settings struct {
	annotation,
	label *[]string
	image,
	resultFileImageDigest,
	resultFileImageSize string
}

func getAnnotation() []string {
	var annotation []string

	if flagValues.annotation != nil {
		return append(annotation, *flagValues.annotation...)
	}

	return annotation
}

func getLabel() []string {
	var label []string

	if flagValues.annotation != nil {
		return append(label, *flagValues.label...)
	}

	return label
}

var flagValues settings

func initializeFlag() {
	// Main flags for the image mutate step to define the configuration, for example
	// the flag `image` will always be used.
	pflag.StringVar(&flagValues.image, "image", "", "The name of image in container registry")
	flagValues.annotation = pflag.StringArray("annotation", nil, "New annotations to add")
	flagValues.label = pflag.StringArray("label", nil, "New labels to add")
	pflag.StringVar(&flagValues.resultFileImageDigest, "result-file-image-digest", "", "A file to write the image digest to")
	pflag.StringVar(&flagValues.resultFileImageSize, "result-file-image-size", "", "A file to write the image size to")
}

func main() {
	if err := Execute(context.Background()); err != nil {
		exitcode := 1

		switch err := err.(type) {
		case *ExitError:
			exitcode = err.Code
		}

		log.Print(err.Error())
		os.Exit(exitcode)
	}
}

// Execute performs flag parsing, input validation and the image mutation
func Execute(ctx context.Context) error {
	initializeFlag()
	pflag.Parse()

	return runMutateImage(ctx)
}

func runMutateImage(ctx context.Context) error {
	annotation := getAnnotation()
	label := getLabel()

	if flagValues.image == "" {
		return &ExitError{Code: 100, Message: "the 'image' argument must not be empty"}
	}

	options := getOptions(ctx)
	ref := flagValues.image

	if len(annotation) != 0 {
		desc, err := crane.Head(ref, *options...)
		if err != nil {
			return fmt.Errorf("checking %s: %v", ref, err)
		}

		if desc.MediaType.IsIndex() {
			return fmt.Errorf("mutating annotations on an index is not yet supported")
		}
	}

	img, err := crane.Pull(ref, *options...)
	if err != nil {
		return fmt.Errorf("pulling %s: %v", ref, err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("getting config: %v", err)
	}
	cfg = cfg.DeepCopy()

	// Set labels.
	if cfg.Config.Labels == nil {
		cfg.Config.Labels = map[string]string{}
	}

	labels, err := splitKeyVals(label)
	if err != nil {
		return err
	}

	for k, v := range labels {
		cfg.Config.Labels[k] = v
	}

	annotations, err := splitKeyVals(annotation)
	if err != nil {
		return err
	}

	// Mutate and write image.
	img, err = mutate.Config(img, cfg.Config)
	if err != nil {
		return fmt.Errorf("mutating config: %v", err)
	}

	img = mutate.Annotations(img, annotations).(containerreg.Image)

	digest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("digesting new image: %v", err)
	}

	r, err := name.ParseReference(ref)
	if err != nil {
		return fmt.Errorf("parsing %s: %v", ref, err)
	}

	if _, ok := r.(name.Digest); ok {
		ref = r.Context().Digest(digest.String()).String()
	}

	if err := crane.Push(img, ref, *options...); err != nil {
		return fmt.Errorf("pushing %s: %v", ref, err)
	}

	fmt.Printf(
		"The image %s was mutated successfully. The new digest is: %s.\n",
		flagValues.image, r.Context().Digest(digest.String()),
	)

	// Writing image digest to file
	if resultFileImageDigest := flagValues.resultFileImageDigest; resultFileImageDigest != "" {
		if err := ioutil.WriteFile(
			resultFileImageDigest, []byte(digest.String()), 0644,
		); err != nil {
			return err
		}
	}

	// Writing image size in bytes to file
	if resultFileImageSize := flagValues.resultFileImageSize; resultFileImageSize != "" {
		size, err := GetCompressedImageSize(img)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(
			resultFileImageSize, []byte(strconv.FormatInt(size, 10)), 0644,
		); err != nil {
			return err
		}
	}

	return nil
}

// GetCompressedImageSize calculate the compressed size of the image.
// By adding up the config and layer sizes we will get the
// total compressed size of the image
func GetCompressedImageSize(img containerreg.Image) (int64, error) {
	manifest, err := img.Manifest()
	if err != nil {
		return 0, err
	}

	configSize := manifest.Config.Size

	var layersSize int64
	for _, layer := range manifest.Layers {
		layersSize += layer.Size
	}

	return layersSize + configSize, nil
}

// splitKeyVals splits key value pairs which is in form hello=world
func splitKeyVals(kvPairs []string) (map[string]string, error) {
	m := map[string]string{}

	for _, l := range kvPairs {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) == 1 {
			return nil, fmt.Errorf("parsing label %q, not enough parts", l)
		}
		m[parts[0]] = parts[1]
	}

	return m, nil
}

func getOptions(ctx context.Context) *[]crane.Option {
	var options []crane.Option

	options = append(options, crane.WithContext(ctx))

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: false}

	var rt http.RoundTripper = transport
	// Add any http headers if they are set in the config file.
	cf, err := config.Load(os.Getenv("DOCKER_CONFIG"))
	if err != nil {
		log.Printf("failed to read config file: %v", err)
	} else if len(cf.HTTPHeaders) != 0 {
		rt = &headerTransport{
			inner:       rt,
			httpHeaders: cf.HTTPHeaders,
		}
	}

	options = append(options, crane.WithTransport(rt))

	return &options
}
