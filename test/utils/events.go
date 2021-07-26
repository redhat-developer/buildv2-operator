package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (t *TestBuild) GetEventsForObject(namespace string, obj runtime.Object) ([]corev1.Event, error) {
	events, err := t.Clientset.CoreV1().Events(namespace).Search(t.Scheme, obj)
	if err != nil {
		return nil, err
	}
	return events.Items, nil
}
