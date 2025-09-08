package common

import "fmt"

func SliceToStringSlice[T fmt.Stringer](slice []T) []string {
	result := make([]string, len(slice))
	for i, val := range slice {
		result[i] = val.String()
	}

	return result
}
