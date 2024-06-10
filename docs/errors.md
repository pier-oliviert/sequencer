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

## Networking Errors
## Workspace Errors


## System Errors
These errors most likely happened outside the operator's scope. Examples of a system error would be a Kubernetes' node error, someone manually deleting watched resources through `kubectl`, etc. It's still useful to have those listed here as that might give an insight to the user as to what has caused the issue.

|E#Number|Title|Description|
|:----|-|-|
|5001|*Could not retrieve a Kubernetes resource*|The operator tried to retrieve a resource that exists in etcd but there was an error that prevented Kubernetes to return the object. The operator filters out NotFoundError(404) so if you see this error, it most likely means something happened.|
