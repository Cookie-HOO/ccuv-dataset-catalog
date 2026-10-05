// Package catalog implements the wire contract for CCUV's official dataset catalog.
package catalog

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	SignedSchema   = "ccuv.official-dataset-catalog/v1"
	EnvelopeSchema = "ccuv.official-dataset-catalog-envelope/v1"
	Algorithm      = "Ed25519"
	Encoding       = "base64"
)

var (
	ErrInvalidEnvelope  = errors.New("invalid catalog envelope")
	ErrInvalidKey       = errors.New("invalid Ed25519 public key")
	ErrInvalidSignature = errors.New("invalid catalog signature")
)

// Envelope is the signed catalog artifact accepted by CCUV.
type Envelope struct {
	Schema    string          `json:"schema"`
	KeyID     string          `json:"key_id"`
	Signed    json.RawMessage `json:"signed"`
	Signature Signature       `json:"signature"`
}

// Signature describes the encoded detached Ed25519 signature over canonical signed JSON.
type Signature struct {
	Algorithm string `json:"algorithm"`
	Encoding  string `json:"encoding"`
	Value     string `json:"value"`
}

// ParseEnvelope strictly decodes a catalog envelope and rejects unknown or duplicate fields.
func ParseEnvelope(data []byte) (Envelope, error) {
	if _, err := Canonicalize(data); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
	}
	var raw map[string]json.RawMessage
	if err := strictDecode(data, &raw); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
	}
	if !exactKeys(raw, "schema", "key_id", "signed", "signature") {
		return Envelope{}, fmt.Errorf("%w: unexpected fields", ErrInvalidEnvelope)
	}
	var envelope Envelope
	if err := strictDecode(data, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
	}
	if envelope.Schema != EnvelopeSchema || envelope.KeyID == "" || len(envelope.Signed) == 0 {
		return Envelope{}, fmt.Errorf("%w: schema, key_id, or signed body", ErrInvalidEnvelope)
	}
	if envelope.Signature.Algorithm != Algorithm || envelope.Signature.Encoding != Encoding || envelope.Signature.Value == "" {
		return Envelope{}, fmt.Errorf("%w: signature metadata", ErrInvalidEnvelope)
	}
	if err := validateSigned(envelope.Signed); err != nil {
		return Envelope{}, err
	}
	canonicalSigned, err := Canonicalize(envelope.Signed)
	if err != nil {
		return Envelope{}, fmt.Errorf("%w: invalid signed body: %v", ErrInvalidEnvelope, err)
	}
	envelope.Signed = canonicalSigned
	return envelope, nil
}

// Verify checks a catalog envelope against a base64-encoded raw Ed25519 public key.
func Verify(envelope Envelope, publicKeyB64 string) error {
	publicKey, err := DecodePublicKey(publicKeyB64)
	if err != nil {
		return err
	}
	signature, err := base64.StdEncoding.Strict().DecodeString(envelope.Signature.Value)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return ErrInvalidSignature
	}
	canonical, err := Canonicalize(envelope.Signed)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, canonical, signature) {
		return ErrInvalidSignature
	}
	return nil
}

// Sign creates an envelope after canonicalizing the supplied signed body.
// It requires a raw base64-encoded private key and deliberately never generates key material.
func Sign(signedBody []byte, keyID, privateKeyB64 string) ([]byte, error) {
	if keyID == "" {
		return nil, errors.New("key ID is required")
	}
	if err := validateSigned(signedBody); err != nil {
		return nil, err
	}
	privateKey, err := DecodePrivateKey(privateKeyB64)
	if err != nil {
		return nil, err
	}
	canonical, err := Canonicalize(signedBody)
	if err != nil {
		return nil, err
	}
	envelope := Envelope{
		Schema: EnvelopeSchema,
		KeyID:  keyID,
		Signed: canonical,
		Signature: Signature{Algorithm: Algorithm, Encoding: Encoding,
			Value: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, canonical))},
	}
	return json.MarshalIndent(envelope, "", "  ")
}

// DecodePublicKey decodes a base64 raw Ed25519 public key.
func DecodePublicKey(value string) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, ErrInvalidKey
	}
	return ed25519.PublicKey(decoded), nil
}

// DecodePrivateKey decodes a base64 Ed25519 seed or raw private key without
// writing it to disk. A 32-byte seed is expanded only in process memory.
func DecodePrivateKey(value string) (ed25519.PrivateKey, error) {
	decoded, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid Ed25519 signing seed")
	}
	switch len(decoded) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(decoded), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(decoded), nil
	default:
		return nil, errors.New("invalid Ed25519 signing seed")
	}
}

// ReadPublicKey reads a text public-key file, allowing surrounding whitespace.
func ReadPublicKey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(data)), nil
}

func validateSigned(data []byte) error {
	var raw map[string]json.RawMessage
	if err := strictDecode(data, &raw); err != nil {
		return fmt.Errorf("invalid signed body: %w", err)
	}
	if !exactKeys(raw, "schema", "catalog_version", "issued_at", "expires_at", "entries") {
		return errors.New("invalid signed body fields")
	}
	if string(raw["schema"]) != `"`+SignedSchema+`"` || string(raw["catalog_version"]) != "1" {
		return errors.New("unsupported signed catalog schema")
	}
	for _, key := range []string{"issued_at", "expires_at", "entries"} {
		if len(raw[key]) == 0 {
			return fmt.Errorf("signed body missing %s", key)
		}
	}
	return nil
}

func exactKeys(value map[string]json.RawMessage, expected ...string) bool {
	if len(value) != len(expected) {
		return false
	}
	for _, key := range expected {
		if _, ok := value[key]; !ok {
			return false
		}
	}
	return true
}

func strictDecode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := ensureEOF(decoder); err != nil {
		return err
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
