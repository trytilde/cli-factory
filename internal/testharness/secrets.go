package testharness

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var (
	ErrNoSecrets       = errors.New("provider test secrets not found")
	ErrSOPSUnavailable = errors.New("sops decrypt unavailable")
)

type SecretsSource string

const (
	SecretsSourceOverride  SecretsSource = "override_test_secrets.yaml"
	SecretsSourceEncrypted SecretsSource = "test_secrets.enc.yaml"
	SecretsSourcePlaintext SecretsSource = "test_secrets.yaml"
)

type Secrets struct {
	Source SecretsSource
	Path   string
	Data   []byte
}

type SecretOptions struct {
	ProviderDir              string
	TestSecretsFile          string
	UseOverrideTestSecrets   bool
	AllowPlaintextTestSecret bool
	Decryptor                Decryptor
}

func SecretOptionsFromEnv(providerDir string) SecretOptions {
	return SecretOptions{
		ProviderDir:            providerDir,
		TestSecretsFile:        os.Getenv("TEST_SECRETS_FILE"),
		UseOverrideTestSecrets: truthy(os.Getenv("USE_OVERRIDE_TEST_SECRETS")),
	}
}

type Decryptor interface {
	Decrypt(ctx context.Context, path string) ([]byte, error)
}

type SOPSDecryptor struct{}

func (SOPSDecryptor) Decrypt(ctx context.Context, path string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "sops", "decrypt", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSOPSUnavailable, stderr.String())
	}
	return out, nil
}

func LoadProviderTestSecrets(ctx context.Context, opts SecretOptions) (Secrets, error) {
	if opts.ProviderDir == "" {
		return Secrets{}, fmt.Errorf("provider dir is required")
	}

	if opts.TestSecretsFile != "" {
		data, err := os.ReadFile(opts.TestSecretsFile)
		if err != nil {
			return Secrets{}, err
		}
		return Secrets{Source: SecretsSourcePlaintext, Path: opts.TestSecretsFile, Data: data}, nil
	}

	overridePath := filepath.Join(opts.ProviderDir, "override_test_secrets.yaml")
	if opts.UseOverrideTestSecrets || fileExists(overridePath) {
		data, err := os.ReadFile(overridePath)
		if err != nil {
			return Secrets{}, err
		}
		return Secrets{Source: SecretsSourceOverride, Path: overridePath, Data: data}, nil
	}

	encryptedPath := filepath.Join(opts.ProviderDir, "test_secrets.enc.yaml")
	if fileExists(encryptedPath) {
		decryptor := opts.Decryptor
		if decryptor == nil {
			decryptor = SOPSDecryptor{}
		}
		decryptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		data, err := decryptor.Decrypt(decryptCtx, encryptedPath)
		if err != nil {
			return Secrets{}, err
		}
		return Secrets{Source: SecretsSourceEncrypted, Path: encryptedPath, Data: data}, nil
	}

	plainPath := filepath.Join(opts.ProviderDir, "test_secrets.yaml")
	if opts.AllowPlaintextTestSecret && fileExists(plainPath) {
		data, err := os.ReadFile(plainPath)
		if err != nil {
			return Secrets{}, err
		}
		return Secrets{Source: SecretsSourcePlaintext, Path: plainPath, Data: data}, nil
	}

	return Secrets{}, ErrNoSecrets
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func truthy(value string) bool {
	switch value {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	default:
		return false
	}
}
