// ccuv-catalog signs and verifies CCUV official dataset catalog envelopes.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Cookie-HOO/ccuv-dataset-catalog/internal/catalog"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "verify":
		err = verify(os.Args[2:])
	case "sign":
		err = sign(os.Args[2:])
	case "canonicalize":
		err = canonicalize(os.Args[2:])
	default:
		usage()
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ccuv-catalog:", err)
		os.Exit(1)
	}
}

func verify(arguments []string) error {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	catalogPath := flags.String("catalog", "", "catalog envelope path")
	publicKeyPath := flags.String("public-key", "", "base64 Ed25519 public key file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *catalogPath == "" || *publicKeyPath == "" || flags.NArg() != 0 {
		return errors.New("usage: ccuv-catalog verify --catalog FILE --public-key FILE")
	}
	data, err := os.ReadFile(*catalogPath)
	if err != nil {
		return err
	}
	envelope, err := catalog.ParseEnvelope(data)
	if err != nil {
		return err
	}
	publicKey, err := catalog.ReadPublicKey(*publicKeyPath)
	if err != nil {
		return err
	}
	if err := catalog.Verify(envelope, publicKey); err != nil {
		return err
	}
	fmt.Printf("verified catalog key_id=%s\n", envelope.KeyID)
	return nil
}

func sign(arguments []string) error {
	flags := flag.NewFlagSet("sign", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	bodyPath := flags.String("body", "", "unsigned catalog body path")
	keyID := flags.String("key-id", "", "signing key identifier")
	keychainService := flags.String("keychain-service", "ccuv-dataset-catalog-signing", "macOS Keychain service")
	keychainAccount := flags.String("keychain-account", "", "macOS Keychain account")
	privateKeyEnv := flags.String("private-key-env", "", "protected environment variable containing a base64 Ed25519 seed")
	outputPath := flags.String("out", "", "output envelope path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *bodyPath == "" || *keyID == "" || *outputPath == "" || flags.NArg() != 0 || (*keychainAccount == "" && *privateKeyEnv == "") || (*keychainAccount != "" && *privateKeyEnv != "") {
		return errors.New("usage: ccuv-catalog sign --body FILE --key-id ID (--keychain-account ACCOUNT [--keychain-service SERVICE] | --private-key-env NAME) --out FILE")
	}
	var privateKey string
	var err error
	if *privateKeyEnv != "" {
		privateKey = os.Getenv(*privateKeyEnv)
		if privateKey == "" {
			return errors.New("catalog signing key is unavailable in the protected environment")
		}
	} else {
		privateKey, err = keychainSeed(*keychainService, *keychainAccount)
		if err != nil {
			return err
		}
	}
	body, err := os.ReadFile(*bodyPath)
	if err != nil {
		return err
	}
	envelope, err := catalog.Sign(body, *keyID, privateKey)
	if err != nil {
		return err
	}
	return os.WriteFile(*outputPath, append(envelope, '\n'), 0o644)
}

func keychainSeed(service, account string) (string, error) {
	command := exec.Command("/usr/bin/security", "find-generic-password", "-s", service, "-a", account, "-w")
	output, err := command.Output()
	if err != nil {
		return "", errors.New("catalog signing key is unavailable in the macOS Keychain")
	}
	seed := strings.TrimSpace(string(output))
	if seed == "" {
		return "", errors.New("catalog signing key is unavailable in the macOS Keychain")
	}
	return seed, nil
}

func canonicalize(arguments []string) error {
	flags := flag.NewFlagSet("canonicalize", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	inputPath := flags.String("in", "", "input JSON path")
	outputPath := flags.String("out", "", "output canonical JSON path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if *inputPath == "" || *outputPath == "" || flags.NArg() != 0 {
		return errors.New("usage: ccuv-catalog canonicalize --in FILE --out FILE")
	}
	data, err := os.ReadFile(*inputPath)
	if err != nil {
		return err
	}
	canonical, err := catalog.Canonicalize(data)
	if err != nil {
		return err
	}
	return os.WriteFile(*outputPath, append(canonical, '\n'), 0o644)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ccuv-catalog <verify|sign|canonicalize> [options]")
}
