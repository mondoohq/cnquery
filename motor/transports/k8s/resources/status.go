package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Status(obj runtime.Object) (string, string) {
	switch x := obj.(type) {
	case *corev1.Pod:
		for i := range x.Status.Conditions {
			if x.Status.Conditions[i].Type == corev1.PodReady {
				return string(x.Status.Conditions[i].Status), x.Status.Conditions[i].Reason
			}
		}
	case *appsv1.Deployment:
		for i := range x.Status.Conditions {
			return string(x.Status.Conditions[i].Status), x.Status.Conditions[i].Reason
		}
	case *appsv1.ReplicaSet:
		for i := range x.Status.Conditions {
			return string(x.Status.Conditions[i].Status), x.Status.Conditions[i].Reason
		}
	default:
		panic(fmt.Errorf("could not access status for resource: %v", x))
	}

	return "", ""
}
