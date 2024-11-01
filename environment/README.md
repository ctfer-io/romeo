# Romeo environment

Deploy a Romeo environment from an Action, or manually.

It deploys Kubernetes resources required to extract the Go coverages of binaries under tests/IVV, given the following architecture.

<div align="center">
    <img src="architecture.excalidraw.png" alt="Romeo environment Kubernetes architecture" width="600px">
</div>

## Usage

### GitHub Actions

To deploy a Romeo environment from an Action, use `ctfer-io/romeo/environment`.
It will create the Kubernetes resources. We recommend you deploy a [Romeo install](../install) per workflow run.

```yaml
      - name: Romeo environment
        id: env
        uses: ctfer-io/romeo/environment@v1
        with:
          kubeconfig: ${{ steps.install.outputs.kubeconfig }}
          namespace: ${{ steps.install.outputs.namespace }}
```

Once your tests ran, you can [download the coverages](../download).

At the end of the Action, it will delete the deployed resources.

#### Inputs

| Name | Type | Default | Description |
|---|---|---|---|
| `kubeconfig` | String |  | **Required.** The kubeconfig to use for deploying a Romeo environment. |
| `namespace` | String |  | The namespace in which to deploy, in case the kubeconfig has access to many. |
| `tag` | String | `latest` | **Required.** The [Romeo webserver docker tag](https://hub.docker.com/r/ctferio/romeo/tags) to use. |
| `claim-name` | String |  | If specified, turns on Romeo's coverage export in the given PersistenVolumeClaim name. This should only be used by CTFer.io to test Romeo itself. |

#### Outputs

| Name | Type | Description |
|---|---|---|
| `port` | String | The port to reach out the Romeo webserver. |
| `claim-name` | String | The PersistentVolumeClaim name for binaries to mount in order to write coverage data. |
| `namespace` | String | The namespace in which Romeo has been deployed. Reuse it to target the PersistentVolumeClaim corresponding to the claim-name. |

### Manually

You may want to deploy the "Romeo environment" to test things manually, or from a non-supported CI system (e.g. GitLab, Drone, Travis).

It has the advantage of not requiring an extensive install of Romeo. We still recommend you **run one Romeo environment per workflow run** to ensure proper isolation between multiple runs thus avoid falsing your coverage measurements.

```bash
# Get in deploy directory
cd deploy

# Create stack and configure
export PULUMI_CONFIG_PASSPHRASE="some-secret"
pulumi stack init --secrets-provider passphrase --stack dev
pulumi config set --secret kubeconfig "$(cat ~/.kube/config)"
pulumi config set namespace ""

# Deploy
pulumi up -y
```
