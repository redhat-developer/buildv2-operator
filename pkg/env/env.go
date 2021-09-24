package env

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// MergeEnvVars merges one slice of corev1.EnvVar into another slice of corev1.EnvVar
// if overwriteValues is false, this function will return an error if a duplicate EnvVar name is encountered
// if overwriteValues is true, this function will overwrite the existing value with the new value if a duplicate is encountered
func MergeEnvVars(new []corev1.EnvVar, into []corev1.EnvVar, overwriteValues bool) ([]corev1.EnvVar, error) {
	// if new, into, or both are empty, there is no need to run through the processing logic
	// just quickly return the appropriate value
	if len(new) == 0 && len(into) == 0 {
		return []corev1.EnvVar{}, nil
	} else if len(new) == 0 {
		return into, nil
	} else if len(into) == 0 {
		return new, nil
	}

	// create a map of the original (into) env vars with the name as the key and
	// their index as the value so we can do value replacements later if overwriteValues is true
	originalEnvs := make(map[string]int)

	for i, o := range into {
		originalEnvs[o.Name] = i
	}

	// merge the new env vars into the original env vars list following a few simple rules
	// based on if the name already exists and whether overwriteValues is true or false
	for _, n := range new {
		_, exists := originalEnvs[n.Name]

		switch {
		case exists && overwriteValues:
			into[originalEnvs[n.Name]] = n
		case exists && !overwriteValues:
			return []corev1.EnvVar{}, fmt.Errorf("environment variable %q already exists", n.Name)
		default:
			into = append(into, n)
		}
	}

	return into, nil
}
