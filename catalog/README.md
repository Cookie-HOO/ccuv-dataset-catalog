# Published catalog

`catalog/catalog.json` is the production artifact consumed from the fixed official raw-GitHub URL by CCUV. It is an Ed25519-signed envelope generated from the reviewable [`signed.json`](signed.json); every change to the signed body must be re-signed with the official key and verified before publication.
