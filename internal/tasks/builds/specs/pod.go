package specs

import (
	"fmt"

	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/env"

	"k8s.io/client-go/tools/record"

	sequencer "se.quencer.io/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodDeployment struct {
	client.Client
	record.EventRecorder
}

var serviceAccountName = env.GetString("CONTROLLER_SERVICE_ACCOUNT", "sequencer-controller-manager")

func PodFor(build *sequencer.Build) *core.Pod {
	pod := &core.Pod{
		ObjectMeta: meta.ObjectMeta{
			Namespace:    build.Namespace,
			GenerateName: fmt.Sprintf("build-%s-", build.Name),
			Annotations: map[string]string{
				"ad.datadoghq.com/build.logs": fmt.Sprintf(`
				[{"source": "go", "service": "sequencer.build", "tags": ["%s"]}]
				`, build.Name),
			},
			OwnerReferences: []meta.OwnerReference{
				{
					APIVersion: build.APIVersion,
					Kind:       build.Kind,
					Name:       build.Name,
					UID:        build.UID,
				},
			},
		},
		Spec: core.PodSpec{
			RestartPolicy:         core.RestartPolicyNever,
			ServiceAccountName:    serviceAccountName,
			ShareProcessNamespace: new(bool),
			Affinity:              build.Spec.Runtime.Affinity,
		},
	}

	*pod.Spec.ShareProcessNamespace = true

	return pod
}
