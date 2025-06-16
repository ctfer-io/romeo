# Romeo webserver

The Romeo webserver is a simplistic application that reuses the Go tooling for coverages introduced in 1.20.
It industrialize the [code coverage for Go integration tests](https://go.dev/blog/integration-test-coverage) blog post and [doc page](https://go.dev/doc/build-cover).

The idea is very simple: on the `/api/v1/coverout` endpoint, it triggers the Go coverage merge command, then zip this, encode base64 to ensure it could be sent on HTTP without losses.
By running it as close to the integration binaries that are tested as possible, you expose those coverages to distant sources without relying on networked files, etc.

On the consumer POV, you only need to call this API, decode base 64, unzip and use. That's it.

## Usage

We recommend you use the Romeo webserver as part of the [Romeo environment](../environment) action.
Nevertheless, as we do not plan supporting another CI system, you are free to reuse our work to cover this use case: be creative !

## Security

### Signature and Attestations

For deployment purposes (and especially in the deployment case of Kubernetes), you may want to ensure the integrity of what you run.

The release assets are SLSA 3 and can be verified using [slsa-verifier](https://github.com/slsa-framework/slsa-verifier) using the following.

```bash
slsa-verifier verify-artifact "<path/to/release_artifact>"  \
  --provenance-path "<path/to/release_intoto_attestation>"  \
  --source-uri "github.com/ctfer-io/romeo" \
  --source-tag "<tag>"
```

The Docker image is SLSA 3 and can be verified using [slsa-verifier](https://github.com/slsa-framework/slsa-verifier) using the following.

```bash
slsa-verifier slsa-verifier verify-image "ctferio/romeo:<tag>@sha256:<digest>" \
    --source-uri "github.com/ctfer-io/romeo" \
    --source-tag "<tag>"
```

Alternatives exist, like [Kyverno](https://kyverno.io/) for a Kubernetes-based deployment.

### SBOMs

A SBOM for the whole repository is generated on each release and can be found in the assets of it.
They are signed as SLSA 3 assets. Refer to [Signature and Attestations](#signature-and-attestations) to verify their integrity.

A SBOM is generated for the Docker image in its manifest, and can be inspected using the following.

```bash
docker buildx imagetools inspect "ctferio/romeo:<tag>" \
    --format "{{ json .SBOM.SPDX }}"
```
