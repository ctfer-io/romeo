# Romeo Install

Install Romeo from an Action, or manually.

It deploys RBAC resources given the following architecture, according to the needs of a [Romeo environment](../environment).

<div align="center">
    <img src="architecture.excalidraw.png" alt="Romeo install Kubernetes architecture" width="600px">
</div>

## Usage

When installing Romeo, the strategy could rather be:
- per organization: not recommend for scalability and stability of parallel infrastructure
- per repository: recommended for scalability and maintainability, with a development or production cluster
- per workflow call: recommended when isolation is required, or when on-the-fly k8s cluster is performed (as [this repository does](../.github/workflows/e2e.yaml))

### GitHub Actions

To deploy a Romeo install from an Action, use `ctfer-io/romeo/install`.
It will create the RBAC resources required by [Romeo environment](../environment) to deploy.

```yaml
      - name: Romeo install
        id: install
        uses: ctfer-io/romeo/install@v1
        with:
          kubeconfig: ${{ secrets.KUBECONFIG }}
```

At the end of the Action, it will delete the deployed resources: it is an ephemeral environment, even within a development or production cluster !

#### Inputs

| Name | Type | Default | Description |
|---|---|---|---|
| `kubeconfig` | String |  | **Required.** The kubeconfig to use for installing Romeo and generating its own kubeconfig (with restreined privileges). |
| `namespace` | String |  | The namespace to install Romeo into. May be randomly generated, as long as it fits Kubernetes naming specification. If not specified, will be randomly generated. |
| `api-server` | String |  | The Kubernetes api-server URL to pipe into the generated kubeconfig. Is inferred from `kubeconfig` whenever possible. Example: "https://cp.my-k8s.lan:6443". |
| `harden` | Bool | false | Whether to harden the namespace or not. Deny all traffic, deny inter-namespace communications, then grant DNS resolution, and grant internet communications. |

#### Outputs

| Name | Type | Description |
|---|---|---|
| `kubeconfig` | String | The kubeconfig to use for deploying a Romeo environment. |
| `namespace` | String | The namespace in which the install has took place. Pass it to Romeo's environment step and tests/IVV ones to know where to deploy too. |

---

### Manually

You may want to deploy the "Romeo install" once for your whole cluster, so _manually_.

It has the advantage of making it only once, then share its instance between all repositories/CI under tests/IVV.
Nevertheless, sharing is not always caring: collisions can happen, and we encourage you to deploy it once per CI run, or at least once per repository. This enables seemless updates between repositories rather than having to sync multiple people.

```bash
# Get in deploy directory
cd deploy

# Create stack and configure
export PULUMI_CONFIG_PASSPHRASE="some-secret"
pulumi stack init --stack dev
pulumi config set --secret kubeconfig "$(cat ~/.kube/config)"
pulumi config set namespace ""  # let it be created
pulumi config set api-server "" # let it be inferred from kubeconfig

# Deploy
pulumi up -y

# Fetch ServiceAccount kubeconfig from outputs
pulumi stack output --show-secrets -j | jq -r '.kubeconfig' > kubeconfig
```
