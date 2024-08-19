# Workspace Specification
```yaml
  networking:
    dns:
      zone: mycoolwebsite.com 
    tunnel:
      cloudflare:
        connector: cloudflared
        accountId: 887766
        route:
          component: clickit
          network: httpserver
        secretKeyRef:
          name: cloudflare-api-token
          key: accessToken
  components:
    - name: redis
      networks:
        - name: redisConn
          port: 6379
          targetPort: 6379
      template:
        containers:
          - name: redis
            image: redis:latest
            ports: 
              - containerPort: 6379
      # [ ... ]
    - name: mysql
      networks:
        - name: mysqlConn
          port: 3306
          targetPort: 3306
      template:
        containers:
          - name: mysql
            image: mysql:latest
            ports: 
              - containerPort: 3306
      # [ ... ]
    - name: clickit
      dependsOn:
        - componentName: mysql
          conditionType: Pod
          conditionStatus: Healthy
        - componentName: redis
          conditionType: Pod
          conditionStatus: Healthy
      template:
        containers:
          - name: click
            image: ${build::clickaroo}
            ports:
              - containerPort: 3000
            env:
              - name: DB_HOST
                value: ${components::mysql.networks.mysqlConn}
              - name: REDIS_HOST
                value: ${components::redis.networks.redisConn}
      networks:
        - name: httpserver
          port: 3000
          targetPort: 3000
      build:
        name: clickaroo
        # [ ... ]
```
<sup>N.B. This is only the `spec` section of the Workspace custom resource definition. A complete [YAML sample is available](../../dev/samples/workspace.yaml) if you're curious as to how it looks like.</sup>

Workspace is the top-level resource where you define a full application to run. While [Build](./build.md) and [Component](./component.md) are somewhat independent and self-contained, Workspaces is the magic glue that makes it possible to define a full application and its dependency and have it deployed as an ephemeral environment.

## Networking
The `networking` section is where the integration with cloud provider happens. Most of Sequencer's lifecycle happens within Kubernetes and is completely isolated from cloud providers. You can imagine this section as the glue between what happens within Kubernetes and the outside world.

### DNS
The `dns` section is a direct integration with `cert-manager`. The dependencies needs to be fully configured before you can generate unique DNS entries for your workspace. The DNS subsection is where you specify high-level information about the workspace you want to create.

|Key|Type|Required|Description|
|:----|-|-|-|
|`zone`|string|✅|Name of the zone you want to run the workspace under. It will create a subdomain off that zone, ie. `pier-olivier.dev` as a zone would create a `workspace-123.pier-olivier.dev` A record and any subdomain specified in this Workspace would be under that dynamically created A record|
|`annotations`|map[string]string|❌|`external-dns` annotations that will be added to the ingress created for the Workspace. This is an escape hatch for you to have control over some behavior, but it should only be used if you know what you're doing. Some annotations can create conflict with Sequencer|

### Tunnels

Tunneling is a great way to test Sequencer on your local machine before investing in cloud provider services. Each tunneling service supported has its own spec for you to use and the documentation on how to use them as a tunnel is documented over there.

|Tunnel Provider|Link|
|:----|-|
|Cloudflare|[Documentation](../tunnels/cloudflare.md)|

### Ingress

The `ingress` section is an abstraction over the [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) API from Kubernetes. It makes the bridge between Sequencer's architectural concepts and core Ingress concepts. Moreover, it uses information from the [DNS](#dns) section to create proper subdomains and TLS certificates so that your application is automatically configured to work with SSL and dynamic DNS.

|Key|Type|Required|Description|
|:----|-|-|-|
|`rules`|[[]RuleSpec](#rulespec-source)|✅|Each rules that describe an endpoint for your application. Those rules makes your application publicly available|
|`className`|string|❌||

#### `RuleSpec` <sup>[[Source]](../../api/v1alpha1/workspaces/ingress.go)</sup>

This spec will generate an `IngressRule` as part of an `Ingress`. That rule will connect directly to one of the component as described in the Workspace.

|Key|Type|Required|Description|
|:----|-|-|-|
|`name`|string|✅|Name of the rule, can be anything, this value is only for giving clarity to each of the rules|
|`component`|string|✅|Name of the component, as defined in the [components' section](#components)|
|`network`|string|✅|Name of the network, as defined within the [components' section](#components)|
|`subdomain`|string|❌|Subdomain off the dynamically created DNS record, ie. `admin`, `web`, `api`, etc.|
|`path`|string|❌|Path to add after the domain for the rule, needs to start with a `/` ie. `/admin`, `/blog`, etc.|

## Components
This contains a list of components that needs to be deployed as part of a workspace. These components can be your own application, requiring an image to be built, but it can also be already built images available publicly like `mysql`, `postgresql`, `redis`, etc. Each component will manage a single pod running the image. You can see a component as a bespoke [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/). It is important to note that it doesn't offer the same guarantees as a Deployment, a Component is not made to run production environments.

The information below regards features that connects Component's together. You should already be familiar with [Component schema](./component.md).

### Dependencies
Dependencies is a way to tell the operator that it should wait to deploy a pod until certain conditions are fullfilled. For a pod on a Component to be deployed, **all dependencies needs to be met**. This is useful when you have an application that needs to connect to a few other components and you need them to be running before deploying the component.

In the example above, the `clickit` component has 2 dependencies: It wants the `Pod` condition for the `mysql` component to equal to `Healthy` and it wants the `Pod` condition for the `redis` component to equal to `Healthy` as well. Only when these 2 components have those conditions met will the pod be deployed for the `clickit` component.

This doesn't mean the component is doing nothing though. Building the image and creating the proper network services will run normally. The component will only wait if it's ready to deploy the final pod and the dependencies aren't met.

### Variable Interpolation

Variable interpolation works similarly to how the dependencies work. Those variables usually points at network service that are managed by sibling components. In the example above, the `clickit` component needs to interpolate 2 variables: The URL that points to the `mysqlConn` network for the component `mysql`, it also needs the URL to the `redisConn` network for the component `redis`.

Because the workspace will dispatch the Components at the same time, those values might not be ready at the same time. The operator is charged to located those values for the Component and won't progress further until all the variable needed are interpolated.
