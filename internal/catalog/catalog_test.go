package catalog

import (
	"strings"
	"testing"
)

const testBody = `{
  "entries": [],
  "issued_at": "2030-01-01T00:00:00Z",
  "catalog_version": 1,
  "schema": "ccuv.official-dataset-catalog/v1"
}`

func TestParseEnvelope(t *testing.T) {
	data := `{"schema":"ccuv.official-dataset-catalog-envelope/v1","key_id":"test-fixture-only","signed":` + testBody + `,"signature":{"algorithm":"Ed25519","encoding":"base64","value":"AA=="}}`
	envelope, err := ParseEnvelope([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(envelope.Signed), `{"catalog_version":1,"entries":[],"issued_at":"2030-01-01T00:00:00Z","schema":"ccuv.official-dataset-catalog/v1"}`; got != want {
		t.Errorf("canonical signed body = %s, want %s", got, want)
	}
}

func TestVerifyRejectsFixtureSignature(t *testing.T) {
	data := `{"schema":"ccuv.official-dataset-catalog-envelope/v1","key_id":"test-fixture-only","signed":` + testBody + `,"signature":{"algorithm":"Ed25519","encoding":"base64","value":"AA=="}}`
	envelope, err := ParseEnvelope([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(envelope, "nWGxne/9Wm7Xco0wZqC4DLWB7pHJWrJvx4OCu0JYbnQ="); err != ErrInvalidSignature {
		t.Fatalf("Verify fixture = %v, want ErrInvalidSignature", err)
	}
}

func TestCanonicalizeRejectsDuplicateKeys(t *testing.T) {
	if _, err := Canonicalize([]byte(`{"schema":"one","schema":"two"}`)); err == nil {
		t.Fatal("Canonicalize accepted duplicate keys")
	}
}

func TestParseEnvelopeRejectsUnknownFields(t *testing.T) {
	data := `{"schema":"ccuv.official-dataset-catalog-envelope/v1","key_id":"test","signed":{"schema":"ccuv.official-dataset-catalog/v1","catalog_version":1,"issued_at":"2030-01-01T00:00:00Z","entries":[]},"signature":{"algorithm":"Ed25519","encoding":"base64","value":"AA=="},"extra":true}`
	if _, err := ParseEnvelope([]byte(data)); err == nil {
		t.Fatal("ParseEnvelope accepted unknown field")
	}
}

func TestCanonicalizeSortsKeys(t *testing.T) {
	canonical, err := Canonicalize([]byte(`{"z":1,"a":"value"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(canonical); got != `{"a":"value","z":1}` {
		t.Errorf("canonical JSON = %s", got)
	}
}

func TestSignRejectsInvalidPrivateKey(t *testing.T) {
	if _, err := Sign([]byte(testBody), "test-key", strings.Repeat("a", 88)); err == nil {
		t.Fatal("Sign accepted invalid private key")
	}
}
