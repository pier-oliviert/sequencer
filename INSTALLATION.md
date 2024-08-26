## How to Install

[Cert-Manager](https://cert-manager.io/) is needed to create ad-hoc certificates for each of the workspace you create. The easiest way to install cert-manager is to follow their [installation guide](https://cert-manager.io/docs/installation/).

```sh
helm repo add jetstack https://charts.jetstack.io --force-update
helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set crds.enabled=true
```

For most installation, you'll also need an Ingress Controller (Gateway API coming soon). The default is the [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/deploy/#quick-start) that can be installed:

```sh
helm upgrade --install ingress-nginx ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx --create-namespace
```

Once you have installed the dependencies above, you can go ahead and install Sequencer's helm chart:

```sh
helm upgrade --install sequencer https://github.com/pier-oliviert/sequencer/releases/download/v0.1/sequencer-0.1.0.tgz \
  --namespace sequencer-system \
  --create-namespace
```

If you plan on using a managed Kubernetes cluster, there's documentation on making sure everything is configured to use Sequencer, if documentation is missing for the managed solution you want to use, please [file an issue](https://github.com/pier-oliviert/sequencer/issues).


**The installation is not complete at this stage**, you'll need to create a `values.yaml` file that will include information that only you can provide to finalize the installation so it can work with the cloud provider you run your cluster in.

### Configure Sequencer with `values.yaml`

Here's a few exemple that represents some real-world scenarios. These aren't exclusive but rather are a way to get started with configuring a Sequencer's helm chart to work with known cloud providers. The full definition of all the helm options is available at the [Helm's documentation page](docs/helm.md).

#### Cloudflare

```yaml
dnsController:
  providerName: cloudflare
  env:
    - name: CF_API_TOKEN
      valueFrom:
        secretKeyRef:
          name: cloudflare-api-token
          key: apiKey
    - name: CF_ZONE_ID
      value: aaaaaaabbbbbbbbbcccccccc

certManager:
  email: youremail@example.com
```

Cloudflare's API requires the API token that can be found on your [profile's page](https://dash.cloudflare.com/profile/api-tokens). If you don't know how to create the `cloudflare-api-token` secret on Kubernetes, you can read how to do that in our [Cloudflare's page](docs/providers/cloudflare.md).

The `CF_ZONE_ID` is the Zone ID you can find on Cloudflare for the domain you'll want to use. For instance, if you were to use the domain `mycooldomain.com` and your DNS Nameserver for that domain is Cloudflare, you'd need to retrieve the Zone ID for `mycooldomain.com` on your Cloudflare's dashboard page and put it there. You can store that value in the same secret you used for the API if you wish, but you'll need to replace the `value` key with a `secretKeyRef`.

Cert-manager requires an email address to [create account](https://cert-manager.io/docs/tutorials/acme/nginx-ingress/). Let's Encrypt will lazily create an account for the email you provide.

Once your `values.yaml` file is fully configured with the proper information, you can update your Helm installation of Sequencer:

```sh
helm upgrade --install sequencer https://github.com/pier-oliviert/sequencer/releases/download/v0.1/sequencer-0.1.0.tgz \
  --namespace sequencer-system \
  --values values.yaml
```

#### AWS' EKS (And Route53)

EKS requires quite a lot of permissions updates to make it work properly. While the `values.yaml` values are easy to find, you need to make sure that EKS has the proper permissions set up with Sequencer's service account so it can talk to Route53. You should look at how to [configure your service account to talk to AWS](docs/providers/eks.md).

```yaml
dnsController:
  providerName: aws
  serviceAccount:
    annotations:
      eks.amazonaws.com/role-arn: arn:aws:iam::444444444444:role/Sequencer-DNS-ServiceAccount
  env:
    - name: AWS_ZONE_ID
      value: Z1111111111111
    - name: AWS_HOSTED_ZONE_ID
      value: Z11111111111P
    - name: AWS_LOAD_BALANCER_HOST
      value: aaaaaaaaaaaaaaaaaaaaaa111111111d-eeeeeeeedddddd50.elb.us-east-2.amazonaws.com

certManager:
  email: youremail@example.com
```

The annotation needs to represent the role you created [here](docs/providers/eks.md).

The environment variables aren't really secrets so they are set as clear text.
|Environment Name|Description|Example value|
|:----|-|-|
|`AWS_ZONE_ID`|The ZONE ID for the domain you want to use with Sequencer, ie. \*.mysuperdomain.com|Z1111111111|
|`AWS_HOSTED_ZONE_ID`|The ZONE ID network load balancer you are going to use to route request with the Ingress Controller. This value can be found on the load balancer's page on AWS|Z1111111111|
|`AWS_LOAD_BALANCER_HOST`|The network load balancer's URL that will be used as an ALIAS for the DNS record. This load balancer is the one that is configured with your ingress controller and that is configured with the `AWS_HOSTED_ZONE_ID`|aaaaaaaaaaaaaaaaaaaaaa111111111d-eeeeeeeedddddd50.elb.us-east-2.amazonaws.com|

