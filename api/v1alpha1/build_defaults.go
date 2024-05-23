package v1alpha1

func (b *Build) Default() {
	if b.Spec.Context == "" {
		b.Spec.Context = "."
	}

	if b.Spec.Args != nil {
		for i := range b.Spec.Args.Items {
			kp := &b.Spec.Args.Items[i]
			kp.Default()
		}
	}

	if b.Spec.Secrets != nil {
		for i := range b.Spec.Secrets.Items {
			kp := &b.Spec.Secrets.Items[i]
			kp.Default()
		}
	}

	for i := range b.Spec.ContainerRegistries {
		cr := &b.Spec.ContainerRegistries[i]
		cr.Default()
	}

	for i := range b.Spec.ImportContent {
		ic := &b.Spec.ImportContent[i]
		ic.Default()
	}
}
