package env

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestMergeEnvVars(t *testing.T) {
	type args struct {
		new             []corev1.EnvVar
		into            []corev1.EnvVar
		overwriteValues bool
	}
	tests := []struct {
		name    string
		args    args
		want    []corev1.EnvVar
		wantErr bool
	}{
		{
			name: "should not fail with nil inputs",
			args: args{
				new:             nil,
				into:            nil,
				overwriteValues: false,
			},
			want:    []corev1.EnvVar{},
			wantErr: false,
		},
		{
			name: "duplicate names should fail with overwriteValues false",
			args: args{
				new: []corev1.EnvVar{
					{Name: "TWO", Value: "twoValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: false,
			},
			want:    []corev1.EnvVar{},
			wantErr: true,
		},
		{
			name: "duplicate names should succeed with overwriteValues true",
			args: args{
				new: []corev1.EnvVar{
					{Name: "TWO", Value: "newTwoValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "newTwoValue"},
			},
			wantErr: false,
		},
		{
			name: "non-duplicate should succeed with overwriteValues false",
			args: args{
				new: []corev1.EnvVar{
					{Name: "THREE", Value: "threeValue"},
					{Name: "FOUR", Value: "fourValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: false,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
				{Name: "THREE", Value: "threeValue"},
				{Name: "FOUR", Value: "fourValue"},
			},
			wantErr: false,
		},
		{
			name: "non-duplicate should succeed with overwriteValues true",
			args: args{
				new: []corev1.EnvVar{
					{Name: "THREE", Value: "threeValue"},
					{Name: "FOUR", Value: "fourValue"},
				},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
				{Name: "THREE", Value: "threeValue"},
				{Name: "FOUR", Value: "fourValue"},
			},
			wantErr: false,
		},
		{
			name: "empty new should return into",
			args: args{
				new: []corev1.EnvVar{},
				into: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
			},
			wantErr: false,
		},
		{
			name: "empty into should return new",
			args: args{
				new: []corev1.EnvVar{
					{Name: "ONE", Value: "oneValue"},
					{Name: "TWO", Value: "twoValue"},
				},
				into:            []corev1.EnvVar{},
				overwriteValues: true,
			},
			want: []corev1.EnvVar{
				{Name: "ONE", Value: "oneValue"},
				{Name: "TWO", Value: "twoValue"},
			},
			wantErr: false,
		}, {
			name: "empty new and into should return empty",
			args: args{
				new:             []corev1.EnvVar{},
				into:            []corev1.EnvVar{},
				overwriteValues: true,
			},
			want:    []corev1.EnvVar{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeEnvVars(tt.args.new, tt.args.into, tt.args.overwriteValues)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeEnvVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}
