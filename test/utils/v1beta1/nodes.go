// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apply "k8s.io/client-go/applyconfigurations/core/v1"
)

// GetNodes returns all Nodes for the TestBuild object
func (t *TestBuild) GetNodes() (*corev1.NodeList, error) {
	client := t.Clientset.CoreV1().Nodes()
	nodes, err := client.List(t.Context, metav1.ListOptions{})
	return nodes, err
}

// AddNodeTaint sets a taint on the given Node name
func (t *TestBuild) AddNodeTaint(nodeName string, taint *corev1.Taint) error {
	client := t.Clientset.CoreV1().Nodes()
	taintApplyCfg := apply.Taint()
	taintApplyCfg.WithKey(taint.Key)
	taintApplyCfg.WithValue(taint.Value)
	taintApplyCfg.WithEffect(taint.Effect)
	nodeSpecApplyCfg := apply.NodeSpec()
	nodeSpecApplyCfg.WithTaints(taintApplyCfg)
	applyCfg := apply.Node(nodeName)
	applyCfg.WithSpec(nodeSpecApplyCfg)
	_, err := client.Apply(t.Context, applyCfg, metav1.ApplyOptions{FieldManager: "application/apply-patch-1}", Force: true})
	if err != nil {
		return err
	}
	return nil
}

// RemoveNodeTaints removes the specified taint on the given Node name
func (t *TestBuild) RemoveNodeTaints(nodeName string) error {
	client := t.Clientset.CoreV1().Nodes()

	// explicitly set taints to null, instead of using an apply config which marshals to empty string values for the taint values.
	// empty string values will fail to validate.
	body := "{\"spec\":{\"taints\":null}}"
	_, err := client.Patch(t.Context, nodeName, types.StrategicMergePatchType, []byte(body), metav1.PatchOptions{FieldManager: "application/apply-patch-2"})
	if err != nil {
		return err
	}
	return nil
}
