# Key policy

This repository contains no private key and does not generate one. Keep signing keys in an approved secret manager or CI secret store. Commit only a production public key after its key ID and CCUV consumer configuration have been reviewed together.

The `sign` command accepts a base64 raw Ed25519 private key only through an explicitly named environment variable. It never creates, prints, or persists key material.
