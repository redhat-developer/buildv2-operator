// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/pflag"

	"github.com/shipwright-io/build/pkg/bundle"
)

var flagValues struct {
	image       string
	target      string
	secretPath  string
	imageDigest string
}

func init() {
	pflag.StringVar(&flagValues.image, "image", "", "Location of the bundle image (mandatory)")
	pflag.StringVar(&flagValues.target, "target", "/workspace/source", "The target directory to place the code")
	pflag.StringVar(&flagValues.imageDigest, "result-file-image-digest", "", "A file to write the image digest")
	pflag.StringVar(&flagValues.secretPath, "secret-path", "", "A directory that contains access credentials (optional)")
}

func main() {
	if err := Do(context.Background()); err != nil {
		log.Fatal(err.Error())
	}
}

// Do is the main entry point of the bundle command
func Do(ctx context.Context) error {
	pflag.Parse()

	if flagValues.image == "" {
		return fmt.Errorf("mandatory flag --image is not set")
	}

	ref, err := name.ParseReference(flagValues.image)
	if err != nil {
		return err
	}

	auth, err := resolveAuthBasedOnTarget(ref)
	if err != nil {
		return err
	}

	log.Printf("Pulling image %q", ref)

	img, err := bundle.PullAndUnpack(
		ref,
		flagValues.target,
		remote.WithContext(ctx),
		remote.WithAuth(auth))
	if err != nil {
		return err
	}

	log.Printf("Image content was extracted to %s\n", flagValues.target)

	digest, err := (*img).Digest()
	if err != nil {
		return fmt.Errorf("digesting new image: %v", err)
	}

	if flagValues.imageDigest != "" {
		if err = ioutil.WriteFile(flagValues.imageDigest, []byte(digest.String()), 0644); err != nil {
			return err
		}
	}

	return nil
}

func resolveAuthBasedOnTarget(ref name.Reference) (authn.Authenticator, error) {
	// In case no secret is mounted, use anonymous
	if flagValues.secretPath == "" {
		log.Printf("No access credentials provided, using anonymous mode")
		return authn.Anonymous, nil
	}

	// Read the registry credentials from the well-known location
	file, err := os.Open(filepath.Join(flagValues.secretPath, ".dockerconfigjson"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cf, err := config.LoadFromReader(file)
	if err != nil {
		return nil, err
	}

	// Look-up the respective registry server inside the credentials
	registryName := ref.Context().RegistryStr()
	if registryName == name.DefaultRegistry {
		registryName = authn.DefaultAuthKey
	}

	authConfig, err := cf.GetAuthConfig(registryName)
	if err != nil {
		return nil, err
	}

	// Return an error in case the credentials do not match the desired
	// registry and list all servers that actually are available
	if authConfig.ServerAddress != registryName {
		var servers []string
		for name := range cf.GetAuthConfigs() {
			servers = append(servers, name)
		}

		return nil, fmt.Errorf("failed to find registry credentials for %s, credentials are available for: %s",
			registryName,
			strings.Join(servers, ", "),
		)
	}

	log.Printf("Using provided access credentials for %s", registryName)
	return authn.FromConfig(authn.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}), nil
}
