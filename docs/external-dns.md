# External-dns for Sequencer

While external-dns has an amazing [tutorial's page](https://kubernetes-sigs.github.io/external-dns/v0.14.2/tutorials/ANS_Group_SafeDNS/) to get you started, there are still some things that are Sequencer specific to make external-dns ready to use.

*This guide assumes you installed external-dns using Helm*.

Because of how Sequencer uses external DNS, some values needs to be set as part of your Helm `values.yaml`. If you installed external-dns already, you can modify the YAML file and update your installation by running

```sh
helm upgrade --install external-dns external-dns/external-dns --values values.yaml
```

The following values needs to be added to your YAML file:

```yaml
policy: sync
interval: 10s
sources:
  - service
  - ingress
  - crd
```

|Key|value|Description|
|:----|-|-|-|
|`policy`|sync|Sync means external-dns will not only create DNS record but it will also delete them when resources in the cluster are deleted|
|`interval`|10s|This can be tweaked, the default is 2 minutes and feels a bit long to wait for DNS to be created every two minutes. 30s might be good a good tradeoff too|
|`sources`|service, ingress, crd|This list doesn't include `crd` by default but Sequencer uses external-dns' custom resource in some places|