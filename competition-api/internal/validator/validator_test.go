package validator

import "encoding/base64"

func base64String(length int) string {
	arr := make([]byte, length)
	for i := range arr {
		arr[i] = 'a'
	}
	return base64.StdEncoding.EncodeToString(arr)
}
