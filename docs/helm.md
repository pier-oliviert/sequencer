# Helm Chart options

Each of the pods/components that make Sequencer can be configured through the Helm chart values chart.

If you'd like an option that isn't currently available, please [file an issue](https://github.com/pier-oliviert/sequencer/issues).

|Key|Description|
|:----|-|-|
|`distribution.image`|Image to use for [distribution](https://github.com/distribution/distribution)|
|`distribution.buildCache.replicas`|The replica count for the build cache deployment|
|`distribution.buildCache.resources`|The resources quotas specified for the build cache deployment|
|`distribution.dockerCache.replicas`|The number of replica for the docker cache deployment|
|`distribution.dockerCache.resources`|The resources quotas specified for the docker cache deployment|
|||
|`sequencer.image`|Image to use for Sequencer|
|`sequencer.pullPolicy`|The pull policy for the image|
|`sequencer.resources`|The resource quotas|
|`sequencer.replicas`|Replica count|
|`sequencer.serviceAccount.annotations`|Annotations for the service account used by Sequencer|
|||
|`builder.image`|Image to use for the builder|
|`builder.pullPolicy`|PullPolicy for builder|
|`builder.buildkitVersion`|The [Buildkit](https://docs.docker.com/build/buildkit/) version to use|
|||
|`solver.image`|Image to use for the cert-manager's solver|
|`solver.pullPolicy`|Pull policy for the image|
|`solver.replicas`|Replica count|
|`solver.privateKeySecretRef.name`|The name of the secret used by Cert-Manager to store the private key. Need to be the same as `certManager.privateKeySecretRef.name`|
|||
|`dns.image`|Image to use for dns controller|
|`dns.pullPolicy`|Pull policy for the DNS' controller|
|`dns.providerName`|Name of the provider, ie. cloudflare, aws|
|`dns.serviceAccount.annotations`|Annotation for the service account. Useful for [EKS](./providers/eks.md)|
|`dns.env`|Environment variables, used to set values for the provider|
|||
|`certManager.serviceAccount.name`|The service account name cert-manager is configured with|
|`certManager.serviceAccount.namespace`|The service account namespace cert-manager is configured with|
|`certManager.server`|Let's Encrypt server. Can be set to staging if needed|
|`certManager.email`|Your email address for Let's Encrypt account, it will be lazily created if it's a new email account|
|`certManager.privateKeySecretRef.name`|Secret name where cert-manager will store the private key ref. This secret will be created by cert-manager lazily|
