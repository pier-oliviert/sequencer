# Container's cache layer with Sequencer

Sequencer deploys [distribution](https://github.com/distribution/distribution) to manage image cache for both the build system and official registries (docker) as it brings many advantages:

- Reduce cost for user; since everything is in-cluster, there's no bandwidth fee, nor is there storage fees
- Builds are much faster as there's minimal roundtrip outside the cluster
- Agressive caching for build artifacts
- Avoid rate limiting from third party(docker)

