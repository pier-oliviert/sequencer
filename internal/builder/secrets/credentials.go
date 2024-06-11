package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	buildConfig "se.quencer.io/api/v1alpha1/builds/config"
)

var ErrCredentialsIncomplete = errors.New("E#1013: couldn't create a valid credential")

type Credentials struct {
	AccessKey   string
	SecretToken string
}

func ReadCredentialsFromDir(path string, c *buildConfig.Credentials) (*Credentials, error) {
	cred := Credentials{}

	switch c.AuthScheme {
	case buildConfig.KeyPair:
		data, err := os.ReadFile(filepath.Join(path, *c.Name, "accessKey"))
		if err != nil {
			return nil, fmt.Errorf("E#1014: error while reading the content of the file (%s/accessKey) -> %w", filepath.Join(path, *c.Name), err)
		}

		cred.AccessKey = string(data)

		data, err = os.ReadFile(filepath.Join(path, *c.Name, "secretToken"))
		if err != nil {
			return nil, fmt.Errorf("E#1014: error while reading the content of the file (%s/secretToken) -> %w", filepath.Join(path, *c.Name), err)
		}

		cred.SecretToken = string(data)

	}

	if cred.AccessKey == "" || cred.SecretToken == "" {
		return nil, ErrCredentialsIncomplete
	}

	return &cred, nil
}
