package oci

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"se.quencer.io/internal/builder/secrets"
)

type Keychain map[Domain]Credential

type Domain string

type Credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (kc Keychain) AddCredential(url string, credential *secrets.Credentials) error {

	ref, err := name.ParseReference(url)
	if err != nil {
		return nil
	}

	kc[Domain(ref.Context().String())] = Credential{
		Username: credential.AccessKey,
		Password: credential.SecretToken,
	}

	return nil
}

// Resolve returns an Authenticator that will be used by the container registry
// to authenticate the session.
func (kc Keychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	host := target.String()
	credential, ok := kc[Domain(host)]

	if !ok {
		return nil, fmt.Errorf("couldn't find credentials for the host: %s", host)
	}

	return authn.FromConfig(authn.AuthConfig{
		Username: credential.Username,
		Password: credential.Password,
	}), nil
}
