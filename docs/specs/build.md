# Build Specification

```yaml
name: my-build
target: dockerfile-target
context: myproj
dockerfile: Dockerfile
args:
  items:
    - key: "go-version"
    - key: "app-version"
  valuesFrom:
    configMapRef:
      name: docker-args
secrets:
  items:
    - key: "some-private-key"
  valuesFrom:
    secretRef:
      name: docker-secrets
containerRegistries:
  - url: "{yourname}/{your-repo}"
    tags:
      - latest
      - v1.03
    credentials:
      authScheme: keyPair
      secretRef:
        name: dockerhub-credentials
importContent:
  - path: source
    credentials:
      authScheme: token
      secretRef:
        name: github-credentials
    contentFrom:
      git:
        ref: main
        url: git@github.com:{yourusername}/{yourproject}.git
        depth: 1
  - path: extra-data
    credentials:
      authScheme: token
      secretRef:
        name: github-credentials
    contentFrom:
      git:
        ref: main
        url: git@github.com:{yourusername}/{yourproject}.git
        depth: 1
```
<sup>N.B. This is only the `spec` section of the Build custom resource definition. A complete [YAML sample is available](../../dev/samples/build.yaml) if you're curious as to how it looks like.</sup>

&nbsp;

## Buildkit
You can configure Buildkit to build the image using top-level fields in the Build spec.

|Key|Type|Required|Description|
|:----|-|-|-|
|`name`|string|✅|A name that is unique within a Workspace that will be matched to a container's image. Read more on the [Component spec](./component.md)|
|`context`|string|❌|Defaults to `.`, if you need to use a different value, you can set it here. This is useful when using multiple import content that points to different paths|
|`dockerfile`|string|❌|Defaults to `Dockerfile`, you can specify where the Dockerfile is located. Can be set with a relative path, eg. `source/docker/Dockerfile.dev`|
|`target`|string|❌|If the Dockerfile is configured to use multi stage builds, you can specify which you target with this field|
|`args`|[*DynamicValues*](../../api/v1alpha1/builds/config/dynamic_values.go)|❌|Key/Value to be passed as [build arguments](https://docs.docker.com/build/guide/build-args/). The key specified will be passed as-is as a key for the build argument|
|`secrets`|[*DynamicValues*](../../api/v1alpha1/builds/config/dynamic_values.go)|❌|Key/Value to be mounted as [build secrets](https://docs.docker.com/build/building/secrets/). The ID of the secret will match they name of the key specified.|

&nbsp;

## [`runtime`](../../api/v1alpha1/builds/import_content.go)
A build runs in a normal pod, and some of the settings for that pod are surfaced back to the user. If there's a pod feature you'd like to see added to this runtime section, please create an Issue for it!

|Key|Type|Required|Description|
|:----|-|-|-|
|`image`|string|❌|The builder image to use. This defaults to the environment variable set in the operator's controller pod deployment|
|`affinity`|[*k8s.Affinity*](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity)|❌|If you need to specify where the builds happen, you can set the node affinity to make sure it runs in the nodes that are suitable for your builds|
|`resources`|[*k8s.ResourceRequirements*](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)|❌|You can set resource limits for a build. These limits might cause builds to be scheduled but not running. However, if you run autoscaler groups on builder nodes, you can get finer-grained control using resources and affinity to lower your cost|

The top level fields in a Build spec are fields that are going to be used by the buildkitd engine to build your image.

&nbsp;

## [`importContent`](../../api/v1alpha1/builds/import_content.go)
Import content is the section where you describe all the repository you need to check out for a build to have all the proper data. The `importContent` section contains a *list* to import. Each entry in that list can describe where the code should be checked out to and where your source code lives. Currently, only Git is supported with a SSH private key.

If you'd like to have another source control supported, please file an issue.

|Key|Type|Required|Description|
|:----|-|-|-|
|`path`|string|❌|Relative path where the source code should be checked out to|
|`contentFrom`|[*ImportSource*](../../api/v1alpha1/builds/import_content.go)|✅|The type of version control to use (git)|
|`credentials`|[*Credentials*](../../api/v1alpha1/builds/config/credentials.go)|❌|Credentials to checkout the code. You can leave it out for a public repository. Required for a private repo. The secret needs to be a *private key*|

### [`ImportSource.Git`](../../api/v1alpha1/builds/import_content.go)
|Key|Type|Required|Description|
|:----|-|-|-|
|`ref`|string|✅|Reference to checkout, it can be a SHA or a tag, eg. `main`|
|`url`|string|✅|URL that points to the repository, eg. https://github.com/pier-oliviert/sequencer.git|
|`depth`|integer|❌|Depth to checkout, unless you havea need for it, you can leave this out|

&nbsp;

## [`containerRegistries`](../../api/v1alpha1/builds/container_registry.go)
Section that describes all the container registries you want to export the image to. When an image is built, it will be cached within the cluster, but Kubernetes doesn't have access to that registry when deploying a pod. For that reason, you need to export the image to a container registry.

|Key|Type|Required|Description|
|:----|-|-|-|
|`url`|string|✅|URL for the container registry repository, the registry needs to support the credential auth scheme provided.|
|`tags`|[]string|✅|List of tags for the image|
|`credentials`|[*Credentials*](../../api/v1alpha1/builds/config/credentials.go)|✅|Credentials to authenticate with the container registry|

&nbsp;

## Embedded field types
These objects are embedded in one of the fields described above.

#### [`credentials`](../../api/v1alpha1/builds/config/credentials.go)
|Key|Type|Required|Description|
|:----|-|-|-|
|`authScheme`|string|✅|The type of authentication scheme this credential represents. Can be one of `token`, `keyPair`|
|`secretRef`|[*LocalObjectReference*](../../api/v1alpha1/builds/config/dynamic_values.go)|✅|The reference to a secret that is bound to the same namespace as the operator|

A `token` scheme means that the authentication only requires a single secret token that will be passed to the provider. When this scheme is used, the underlying secret is **required** to have the key `privateKey` set in its data.

A `keyPair` is a set of key that will be used to authenticate. It can be any string value for the pair: a username, email, password, accessKey, secretToken, etc. Because the credentials is passed through the project and is used in several places, it wouldn't be practical to support an arbitrary name for the keys. When you use a `keyPair` scheme, the underlying secret is **required** to include two entries:

 - `accessKey`
 - `secretToken`

If, for example, the credentials you have from your provider is a username/password, you would set the `accessKey` as the username and you would put the password in the `secretToken` field.

#### [`DynamicValues`](../../api/v1alpha1/builds/config/dynamic_values.go)
User provided values stored in either a ConfigMap or a Secret.

|Key|Type|Required|Description|
|:----|-|-|-|
|`valuesFrom`|[*SourceRef*](../../api/v1alpha1/builds/config/dynamic_values.go)|✅|Reference to checkout, it can be a SHA or a tag, eg. `main`|
|`items`|[*[]KeyToPath*](../../api/v1alpha1/builds/config/dynamic_values.go)|✅|List of keys to be passed to the build|

#### [`SourceRef`](../../api/v1alpha1/builds/config/dynamic_values.go)
Exactly one of the reference needs to be specified.

|Key|Type|Required|Description|
|:----|-|-|-|
|`configMapRef`|[*LocalObjectReference*](../../api/v1alpha1/builds/config/dynamic_values.go)|❌|Reference to an existing ConfigMap, in the same namespace|
|`secretRef`|[*LocalObjectReference*](../../api/v1alpha1/builds/config/dynamic_values.go)|❌|Reference to an existing Secret, in the same namespace|

#### [`KeyToPath`](api/v1alpha1/builds/config/dynamic_values.go)
|Key|Type|Required|Description|
|:----|-|-|-|
|`key`|string|✅|Name of the key, as it exists in the data's resource, ie. Secret or ConfigMap|

#### [`LocalObjectReference`](../../api/v1alpha1/builds/config/dynamic_values.go)
LocalObjectReference means the object exists in the local frame of reference, this means the object lives in the same namespace as the operator.

|Key|Type|Required|Description|
|:----|-|-|-|
|`name`|string|✅|Name of the resource, ie. the secret's name|
