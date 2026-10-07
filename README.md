# CCUV Dataset Catalog

A standard-library Go foundation for signing and verifying the CCUV official dataset catalog contract. It ships no private signing material and no production `catalog/catalog.json` yet: the first runtime envelope must be created only after a real immutable dataset release exists. The only catalog envelope under `fixtures/` is deliberately invalid (placeholder signature) and exists solely to prove rejection paths; it cannot be accepted by CCUV.

## Contract

CCUV verifies an envelope with these exact top-level members:

- Envelope schema: `ccuv.official-dataset-catalog-envelope/v1`
- Body schema: `ccuv.official-dataset-catalog/v1`
- Signature: base64-encoded raw `Ed25519` signature
- Signed bytes: RFC 8785 JSON Canonicalization Scheme representation of `signed`

Schemas are versioned in [schemas/](schemas/). Each entry has a localized `title`, a short localized `summary` for browsing, and one canonical `repository`; platform artifacts then describe immutable release assets without repeating the repository. The consumer independently enforces catalog lifetime, exact entry/artifact fields, digest formats, and trusted GitHub release URLs. This foundation does not yet duplicate those full consumer semantic checks, so a catalog must be validated by the released ccuv consumer before signing or publication.

## Tooling

Build and verify the public fixture:

```sh
go run ./cmd/ccuv-catalog verify \
  --catalog fixtures/test-envelope.json \
  --public-key fixtures/test-public-key.txt
```

Canonicalize an unsigned body before review:

```sh
go run ./cmd/ccuv-catalog canonicalize \
  --in template/catalog-body.template.json \
  --out /tmp/catalog-body.json
```

The production public key is committed at [`keys/ccuv-official-2026-01.pub`](keys/ccuv-official-2026-01.pub). Its private seed is held in the maintainer macOS Keychain under service `ccuv-dataset-catalog-signing` and account `ccuv-official-2026-01`; it is also scoped to the protected `catalog-publication` GitHub Environment for the audited publication workflow.

Sign locally from Keychain without exporting the seed:

```sh
go run ./cmd/ccuv-catalog sign \
  --body catalog/signed.json \
  --key-id ccuv-official-2026-01 \
  --keychain-account ccuv-official-2026-01 \
  --out catalog/catalog.json
```

The protected workflow exposes `CATALOG_SIGNING_SEED` only to its `publish-catalog` job. It never writes or prints the seed, and verifies the completed envelope using only the committed public key. Do not use the template as an artifact: it contains placeholder timestamps and lives outside `catalog/catalog.json`. The runtime catalog path must remain absent until a released dataset artifact supplies immutable digest and size metadata.

When release metadata is ready, commit the reviewable `catalog/signed.json` through a protected PR, sign it, validate the envelope with released ccuv, and commit the resulting `catalog/catalog.json` to `main`. Uploading an Actions artifact alone never publishes the fixed raw-GitHub runtime URL.

## CI model

CI needs only the committed public key and the candidate envelope:

```sh
ccuv-catalog verify --catalog catalog/catalog.json --public-key keys/ccuv-official-2026-01.pub
```

The workflow at [.github/workflows/catalog.yml](.github/workflows/catalog.yml) has a public-key-only `verify-catalog` job and a manually dispatched protected `publish-catalog` job. Publication uses the GitHub environment `catalog-publication`; its `CATALOG_SIGNING_SEED` secret is scoped only to that job and its output is verified with `keys/ccuv-official-2026-01.pub` before it is retained. The workflow prepares a reviewed signed artifact; publication to `main` remains a separate reviewed commit until an explicit protected automation path is added. The repository does not create keys or alter GitHub environment/reviewer settings. Until independent reviewers are configured, the required reviewer may be the same maintainer; that is a governance limitation, not an independent approval.

## Development

```sh
gofmt -w ./cmd ./internal
go test ./...
go vet ./...
```

The project has no third-party Go dependencies.
