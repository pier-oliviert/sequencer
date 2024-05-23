# What is Sequencer?
Sequencer is a Open Source Kubernetes [Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) that manages previews of your production application. Users of PaaS like Heroku and Vercel are familiar with the concept of running sequencer environments for testing and QA purposes. You can now to do the same, wherever your infrastructure lives. You can go from zero to a fully deployed application all within Kubernetes.

## ⚠️ Technical preview ⚠️
This project is still very early on and as such, should be considered a technical preview. Building and deploying applications involve many independent features that need to work in concert to bring your application up. Because of the overall complexity, it's expected that you'll hit edge cases along the way and if _when_ do, please open up an Issue.

The goal for Sequencer is to deliver a high quality software. You can [read more about the philosophy behind Sequencer](./PHILOSOPHY.md).

# Features

- Build image with [Buildkit](https://docs.docker.com/build/buildkit/)
- Cache images locally with [Distribution](https://github.com/distribution/distribution)
- Publish images to your Container Registry (Docker, AWS, Google, etc.)
- Deploy your application with all its dependencies
- Create network routes using integrations (Cloudflare, AWS, Google, etc.)
- Create unique URL that points to your deployed application

## Supported cloud providers

|Provider Name|Tunneling|Ingress|DNS|
|:--------|-|-|-|
|Cloudflare|✅|🤞|✅|
|AWS|🤞|🏗️|🏗️|
|Google|🤞|🏗️|🏗️|

Supported: ✅ In the work: 🏗️ Maybe: 🤞

# Install

## Dependencies

You will need to have [Cert-Manager](https://cert-manager.io/) running beforehand. The easiest way to install cert-manager is to follow their [installation guide](https://cert-manager.io/docs/installation/).

## Default Helm installation
Using [Helm](https://helm.sh/), you can install Sequencer as follows:

```sh
helm install sequencer https://github.com/pier-oliviert/sequencer/releases/download/v0.1/sequencer-0.1.0.tgz \
  --namespace sequencer-system \
  --create-namespace
```

## Try it out!

The best way to understand how Sequencer works is to try it out. The [Get Started guide](./GET_STARTED.md) has all the information you need to get a workspace up and running.

## Reference

Sequencer has a few custom resources, the main ones are designed to work like [Matryoshka doll](https://en.wikipedia.org/wiki/Matryoshka_doll). The Workspace is the top level resource, which includes N-number of a Components, where each of those components can have at-most one Build. Each of the resource has its own Spec for you to peruse.

- [Workspace](./docs/specs/workspace.md)
- [Component](./docs/specs/component.md)
- [Build](./docs/specs/build.md)