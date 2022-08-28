package execruntime

const K8S_OPERATOR = "kubernetes"

var kubernetesEnv = &RuntimeEnv{
	Id:        K8S_OPERATOR,
	Name:      "Kubernetes",
	Namespace: "k8s.mondoo.com",
	Prefix:    "KUBERNETES",
	Identify: []Variable{
		{
			Name: "KUBERNETES_ADMISSION_CONTROLLER",
			Desc: "Running from the Mondoo Kubernetes admission controller",
		},
	},
	Variables: []Variable{},
}
