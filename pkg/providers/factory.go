package providers

import (
	"context"
	"fmt"
	"log"
	"os"

	"k8s.io/utils/env"

	sequencer "github.com/pier-oliviert/sequencer/api/v1alpha1"
)

const kProviderName = "DNS_SEQUENCER_PROVIDER_NAME"
const kProviderConfigPath = "/var/run/configs/provider"

type Provider interface {
	Create(context.Context, *sequencer.DNSRecord) error
	Delete(context.Context, *sequencer.DNSRecord) error
}

func NewProvider(name string) (Provider, error) {
	switch name {
	case "cloudflare":
		return NewCloudflareProvider()
	case "aws":
		return NewAWSProvider()
	case "":
		return nil, fmt.Errorf("E#4001: The environment variable %s need to be set with a valid provider name", kProviderName)
	}

	return nil, fmt.Errorf("E#4001: The environment variable %s need to be set with a valid provider name, got %s", kProviderName, name)
}

// Same as NewProvider but throw a fatal exception if
// the configuration settings can't initialize a provider.
func DefaultProvider() Provider {
	value, err := retrieveValueFromEnvOrFile(kProviderName)
	if err != nil {
		log.Fatal(err)
	}

	p, err := NewProvider(value)
	if err != nil {
		log.Fatal(err)
	}
	return p
}

// First check if the environment variable is set, if not, let's look for the
// token at `${kProviderConfigPath}/${kCloudflareAPIKeyName}` and read the content
// of that file into token
func retrieveValueFromEnvOrFile(envNameOrFileName string) (content string, err error) {
	content = env.GetString(envNameOrFileName, "")

	if content == "" {
		path := fmt.Sprintf("%s/%s", kProviderConfigPath, envNameOrFileName)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("E#4002: %s does not exist as an environment variable and a file(%s) with this name could not be found", envNameOrFileName, path)
		}
		content = string(data)
	}

	return content, nil
}
