package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResourceId(t *testing.T) {
	t.Run("namespaced resource", func(t *testing.T) {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "test123",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
		}

		id := objIdFromFields(pod.Kind, pod.Namespace, pod.Name)

		assert.NotEmpty(t, id)
		assert.Equal(t, "pod:test123:nginx", id)
	})

	t.Run("non-namespaced resource", func(t *testing.T) {
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
		}

		id := objIdFromFields(ns.Kind, ns.Namespace, ns.Name)

		assert.NotEmpty(t, id)
		assert.Equal(t, "namespace:nginx", id)
	})
}
