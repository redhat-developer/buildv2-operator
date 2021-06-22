// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

func TestEnv_ValidatePath(t *testing.T) {
	tests := []struct {
		name       string
		env        []corev1.EnvVar
		wantErr    bool
		errReason  string
		errMessage string
	}{
		{
			name: "empty env var name should fail",
			env: []corev1.EnvVar{
				{
					Name:  "",
					Value: "some-value",
				},
			},
			wantErr:    true,
			errReason:  string(build.SpecEnvNameCanNotBeBlank),
			errMessage: "name for environment variable must not be blank",
		},
		{
			name: "empty env var value should fail",
			env: []corev1.EnvVar{
				{
					Name:  "some-name",
					Value: "",
				},
			},
			wantErr:    true,
			errReason:  string(build.SpecEnvValueCanNotBeBlank),
			errMessage: "value for environment variable \"some-name\" must not be blank",
		},
		{
			name: "compliant env var should pass",
			env: []corev1.EnvVar{
				{
					Name:  "some-name",
					Value: "some-value",
				},
			},
			wantErr:    false,
			errReason:  "",
			errMessage: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &build.Build{
				Spec: build.BuildSpec{
					Env: tt.env,
				},
			}
			e := NewEnv(b)
			if err := e.ValidatePath(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("Env.ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if b.Status.Reason != build.BuildReason(tt.errReason) {
				t.Errorf("Build.Status.Reason = %v, wanted: %v", b.Status.Reason, tt.errReason)
			}
			if b.Status.Message != tt.errMessage {
				t.Errorf("Build.Status.Message = %v, wanted: %v", b.Status.Message, tt.errMessage)
			}
		})
	}
}
