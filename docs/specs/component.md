# Component Specification

```yaml
name: click-mania
template:
  containers:
    - name: click
      image: ${build::clickaroo}
      ports:
        - containerPort: 3000
      env:
        - name: DB_HOST
          value: ${components::mysql.networks.tcp}
        - name: REDIS_HOST
          value: ${components::redis.networks.tcp}
        - name: DB_NAME
          value: mydb
        - name: DB_USER
          value: potest
        - name: DB_PASSWORD
          value: whatever
      command:
        - /srv/aurora-test
        - start
networks:
  - name: http
    port: 3000
    targetPort: 3000
dependsOn:
  - componentName: mysql
    conditionType: Pod
    conditionStatus: Healthy
  - componentName: redis
    conditionType: Pod
    conditionStatus: Healthy
build:
  name: clickaroo
  # [ ... ]
```
<sup>N.B. This is only the `spec` section of the Component custom resource definition. A complete [YAML sample is available](../../dev/samples/component.yaml) if you're curious as to how it looks like.</sup>

## Template
The template is imported from core [K8s API](https://kubernetes.io/docs/concepts/workloads/pods/#pod-templates). It is used to create a pod where the operator will inject values so that the component runs the image you built, and can communicate with all the other components you run inside a [Workspace](./workspace.md).

Dynamic values are generated through variable interpolation and only a few areas in a template are you able to use dynamic values:

- `image`: The image name of a container can be replaced by the build's image.
- `env`: Environment variables can have their value replaced by a network value (i.e internal/external URL)

### Variable Interpolations
The operator checks for values that respects the rules of an variable template. To use variable interpolation, you need to use the following structure:

```
${crdType::sourceName[.sectionType.sectionName]}
```

|Name|Required|Description|
|:----|-|-|
|`crdType`|✅|The custom resource type where the value is, can be one of `build`, `components`|
|`sourceName`|✅|Name of the resource as defined in the spec of the custom resource.|
|`sectionType`|❌|Section type within the custom resource, can only be `networks`|
|`sectionName`|❌|Name of the resource within the section type, ie. Name of the network you want to point to|

The interpolation is extracted using a [regexp rule](../../internal/tasks/components/variables.go). 

## Networks
Each network represent a network interface that is made available for either other component (ie. a SQL database needs to have a network connection for your backend to connect to). Network entries need to point to a port defined in the container's template. In the example at the top, the network `http` points to the **containerPort: 3000**.

A network is more than just port mapping, it will create a Kubernetes [Service](https://kubernetes.io/docs/concepts/services-networking/service/) which means it will have an internal DNS entry that can be used by other components to connect to it. The [Workspace](./workspace.md) can be customized to create tunneling that connects to a network entry.

## Dependencies
This list of dependencies the component should wait for **before** creating the pod. Dependency only works for component that lives within the workspace. All the conditions needs to be met before the pod is created.

|Name|Required|Description|
|:----|-|-|
|`componentName`|✅|Name of the component as defined in the spec of that component, eg. `click-mania`|
|`conditionType`|✅|The [condition type](../../api/v1alpha1/components/const.go) it should watch. These are only Component's condition type. `Pod` is usually the type you want to watch|
|`conditionStatus`|✅|[Status](../../api/v1alpha1/conditions/condition.go) of the condition that matches the condition type specified. Should almost always be set to `Healthy`|

## Build
Build represent a [Build custom resource](./build.md) that will be executed as part of this component. The `name` of the build is important, it is how you can reference an image for a container to point to a build. In the example above, the build's name is set as the container's image (`${build::clickaroo}`). When the build is completed, Sequencer will replace that value with the URL pointing at the container registry, including the SHA256 for this specific build.