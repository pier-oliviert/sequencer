# Errors and their meaning

If you had an error starting with `E#[Number]`, and you don't know what to do, you can look the error number here to find more information about it. If the error is not present here or the description is misleading, please write an issue to get it fixed.

Errors that are tagged like this also wrap the underlying error if one exists. Those wrapped error usually provide more context about the origin of the problem. Also, when an error occurs, the error
will be dispatched to the event recorder of the object that owned the process. For instance, if an error occurs while running a task for the Build custom resource, that build will have the error attached to it's event recorder.

## Build Errors
Errors that happened while building an image for a workspace. Most of these errors are related to the [Build Schema](./specs/build.md).

|E#Number|Title|Description|
|:----|-|-|
|1001|*Pod could not be created*|A Build was created, but when it came time to dispatch the pod that runs the actual build, it failed to do so. This can be caused, for example, by a badly configured service account for the operator|
|1002|*ValuesFrom not specified*|In the Build spec, a `DynamicValue` has a misconfigured `ValuesFrom`|
|1003|*ContentFrom is nil*|ContentFrom, in the `ImportContent` section, contains many optional field. One of those field need to be set. If you're using Github, the `git` optional field needs to be configured|
|1004|*ContainerRegistries list is empty*|At least one ContainerRegistry needs to be specified, this is going to be used by the operator to run a pod with the build you created|
|1005|*Builds are immutable*|Something tried to update or patch the Build custom Resource. This operator is not possible as builds wouldn't know what to do.|
|1006|*Invalid secret for the credentials' authScheme*|The secret doesn't fit the format specified by the [`authScheme`](./specs/build.md#credentials) the content of the secret needs to be an exact match as described in the reference|
|1007|*Secret could not be retrieved*|The secret could not be retrieved, the secret's name is set by the user, was it created in the same namespace, ie. `sequencer-system`?|
|1008|*No pod dispatched for the build*|Sequencer tried to dispatch a pod to run the build, but it failed. Could there be a permission issue within Kubernetes? If you don't know how you got there, you can file an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|1009|*Pod had an unexpected failure*|The pod running the build crashed before the pod had time to update the status and conditions of the Build custom resource. This can be caused by a bug with Buildkit. You can look at the logs of the pod to see what happened. If you feel this is a bug with Sequencer, you can create an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|1010|*Expected one build, found more*|In the current state, only one build can be associated to a given Component. However, multiple Build references were found. This is likely a bug, you should file an [issue](https://github.com/pier-oliviert/sequencer/issues).|
|1011|*Pod had an error*|An error occurred while building your image. The operator behave correctly, but the build most likely had an error because of a user error and cannot continue further. You should look at the log for the build pods as it may have the information required to solve the problem|
|1012|*Could not checkout the source repository*|There was an error trying to checkout the source repository. The error attached should give you more details as to what happened|
|1013|*Invalid Credentials*|The credentials were provided but were incomplete|
|1014|*Could not read the content of the secret at file location*|In the build, the secrets provided are mapped to a temporary file created so the build system can safely read those secrets. This error might be an [bug](https://github.com/pier-oliviert/sequencer/issues)|
|1015|*Git error during checkout*|There was an error checking out the code from a git repository. The attached error should provide more information|
|1016|*Wrong auth scheme for source control*|Credentials were provided, but the [`authScheme`](../docs/specs/build.md#importcontent) doesn't match a supported option for the version control system (Github, etc.)|


## Component Errors
Errors related to managing a component.
|E#Number|Title|Description|
|:----|-|-|
|2001|*Could not find variable*|The variable, as defined, doesn't exist. This can happen if there's a typo in the variable's name. For example, if the image name is `${build::appOne}`, and the build for the component is named `appTwo`, this error will be triggered|
|2003|*No pod exists for this component*|No pod was created for the component running. This is most likely a bug and should be reported as an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|2004|*Retrieved more than one pod*|Sequencer expected to retrieve a single pod, but Kubernetes returned more than one. This is most likely an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|2005|*Pod had a failure in one of the container*|Your application was deployed correctly, but crashed. Your application logs in the pod listed should give you better information about your application's failure|
|2006|*Variable could not be parsed*|The variable couldn't be parsed, is likely to be an [issue](https://github.com/pier-oliviert/sequencer/issues) as the value should have been whitelisted prior|
|2007|*No resolver found that matches the type*|The type of resolver currently supported is `build` and `component`. The value provided doesn't match any of them, read more about [variable interpolation](../docs/specs/component.md#variable-interpolations)|
|2008|*Could not find a resource with the label selector provided*|The label selector provided did not match any build. This might be an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|2009|*Could not find a build associated with this component*|The variable referencing a build has a name that couldn't match any build. The name of the build, in the spec, needs to match the build's name in the [variable provided](../docs/specs/component.md#variable-interpolations).|
|2010|*Failed to decode the Index Manifest of the build*|When a build is completed, it will store the Index Manifest describing the image in it's status. The data could not be decoded to interpolate the variable for the component. This is likely an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|2011|*The build referenced doesn't include a valid image*|The Index Manifest store in the Build's status doesn't include a valid image to use. This is likely an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|2012|*The variable doesn't have the right format*|The format doesn't respect the one given in the error message|
|2013|*The variable doesn't include a proper section*|The section is the second part of a variable(eg. `${components::ComponentName.section.serviceName}`), the value provided doesn't match a value supported|


## Workspace Errors
Workspace errors are top level errors that aren't specific to any of the underlying custom resource. This also include errors that can happen at a lower level but are to general to be attributed to any other custom resource.

|E#Number|Title|Description|
|:----|-|-|
|3001|*Failed to parse the label selector*|An error with label selector should not be caused by a user, if this happens to you, please file an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|3002|*Could not create the component*|The workspace tried to create the Component custom resource with the spec provided in the Workspace's spec, but an error occured. This might be an [issue](https://github.com/pier-oliviert/sequencer/issues)|
|3003|*Could not create a DNS record with external-dns*|There was an error creating an external-dns custom resource. The error attached might give you more information|
|3004|*Secret doesn't have a value for the specified key*|The secret referenced doesn't include a value for the key given. The error message should include both the secret's name and the key that it was looking for|
|3007|*Could not create an integration client*|There was an error initializing the integration client|
|3008|*Could not update tunnel with DNS record information*|An error prevented the tunnel to be configured to point to the DNS record|
|3009|*Could not delete the tunnel*|The tunnel could not be deleted from the integration. It may be orphaned on the integration's side|
|3010|*Could not create and configure the tunnel*|There was an error creating the tunnel. It may be orphaned on the integration's side|
|3011|*The DNS Spec doesn't include a valid provider*|The DNS Spec included in the workspace spec does not use a valid provider. The list of provider is available in the [documentation](../docs/specs/workspace.md#networking)|
|3012|*The Tunnel Spec doesn't include a valid provider*|The Tunnel Spec included in the workspace spec does not use a valid provider. The list of provider is available in the [documentation](../docs/specs/workspace.md#networking)|
|3013|*Could not retrieve the load balancer*|The service type=LoadBalancer could not be found matching the reference provided. Make sure it exists and the namespace/name are correct|

## Integration Errors


## System Errors
Errors that are outside the scope of each individual custom resource. These errors could be generic Kubernetes' error or could be errors that are global within the Operator.

|E#Number|Title|Description|
|:----|-|-|
|5001|*Could not retrieve a Kubernetes resource*|The operator tried to retrieve a resource that exists in etcd but there was an error that prevented Kubernetes to return the object. The operator filters out NotFoundError(404) so if you see this error, it most likely means something happened.|
|5002|*Could not retrieve list of resource*|Kubernetes returned an error when trying to retrieve a list of pods for the component. This could indicate that your Kubernetes cluster is unhealthy|
|5003|*Could not unlock the condition*|Each custom resources have a set of conditions that the operator manipulates to reflect the state of the given resource. Usually, the operator will attempt to lock a resource before making changes to it. A failure to lock a condition means that the operator could not start working on the condition specified in the error.|
