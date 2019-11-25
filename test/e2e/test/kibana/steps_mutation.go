// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"
	"testing"

	"github.com/elastic/cloud-on-k8s/pkg/apis/kibana/v1beta1"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/hash"
	"github.com/elastic/cloud-on-k8s/pkg/utils/k8s"
	"github.com/elastic/cloud-on-k8s/test/e2e/test"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func (b Builder) MutationTestSteps(k *test.K8sClient) test.StepList {
	return b.AnnotatePodsWithBuilderHash(k).
		WithSteps(b.UpgradeTestSteps(k)).
		WithSteps(b.CheckK8sTestSteps(k)).
		WithSteps(b.CheckStackTestSteps(k))
}

func (b Builder) AnnotatePodsWithBuilderHash(k *test.K8sClient) test.StepList {
	return []test.Step{
		{
			Name: "Annotate Pods with a hash of their Builder spec",
			Test: test.Eventually(func() error {
				if b.MutatedFrom == nil {
					panic("builder has to have MutatedFrom set if it's a mutation builder")
				}

				var pods corev1.PodList
				if err := k.Client.List(&pods, test.KibanaPodListOptions(b.Kibana.Namespace, b.Kibana.Name)...); err != nil {
					return err
				}

				for _, pod := range pods.Items {
					if pod.Annotations == nil {
						pod.Annotations = make(map[string]string)
					}
					pod.Annotations[BuilderHashAnnotation] = hash.HashObject(b.MutatedFrom.Kibana.Spec)
					if err := k.Client.Update(&pod); err != nil {
						// may error out with a conflict if concurrently updated by the operator,
						// which is why we retry with `test.Eventually`
						return err
					}
					fmt.Printf("pod %s has builder hash set to %s\n", pod.Name, pod.Annotations[BuilderHashAnnotation])
				}
				return nil
			}),
		},
		// make sure this is propagated to the local cache so next test steps can expect annotated pods
		{
			Name: "Wait for annotated Pods to appear in the cache",
			Test: test.Eventually(func() error {
				var pods corev1.PodList
				if err := k.Client.List(&pods, test.KibanaPodListOptions(b.Kibana.Namespace, b.Kibana.Name)...); err != nil {
					return err
				}

				for _, pod := range pods.Items {
					if pod.Annotations[BuilderHashAnnotation] == "" {
						return fmt.Errorf("pod %s is not annotated with %s yet", pod.Name, BuilderHashAnnotation)
					}
				}
				return nil
			}),
		},
	}
}

func (b Builder) MutationReversalTestContext() test.ReversalTestContext {
	panic("not implemented")
}

func (b Builder) UpgradeTestSteps(k *test.K8sClient) test.StepList {
	return test.StepList{
		{
			Name: "Applying the Kibana mutation should succeed",
			Test: func(t *testing.T) {
				var kb v1beta1.Kibana
				require.NoError(t, k.Client.Get(k8s.ExtractNamespacedName(&b.Kibana), &kb))
				kb.Spec = b.Kibana.Spec
				require.NoError(t, k.Client.Update(&kb))
			},
		}}
}
