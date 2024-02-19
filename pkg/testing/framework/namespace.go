package framework

import (
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewEphemeralNamespace() *corev1.Namespace {
	name := NormalizeNameString("t-" + uuid.NewString())
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return ns
}

func NormalizeNameString(name string) string {
	if len(name) > 63 {
		return name[:63]
	}
	return name
}
