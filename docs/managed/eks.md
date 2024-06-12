# Configure EKS for Sequencer

## Add custom resource definitions permissions
To be able use sequencer (or any other operator), you'll need to have the right permission, when adding the helm chart.

group: apiextensions.k8s.io
resource: customresourcedefinitions

You'll need `AmazonEKSClusterAdminPolicy`