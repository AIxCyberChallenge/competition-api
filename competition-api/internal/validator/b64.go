package validator

import (
	"encoding/base64"
)

// ensure the data length is less than the maximum base64 length for a given length without decoding the base64
func validateBase64Len(dataLen int, length int) bool {
	return dataLen <= base64.StdEncoding.EncodedLen(length)
}

// ensures an encoded patch is less than the maximum length for the allowable max patch size
func ValidatePatchSize(dataLen int) bool {
	return validateBase64Len(dataLen, 1<<10*100)
}

// ensures an encoded trigger is less than the maximum length for the allowable max trigger size
func ValidateTriggerSize(dataLen int) bool {
	return validateBase64Len(dataLen, 1<<21)
}

// ensures an encoded free form is less than the maximum length for the allowable max freeform size
func ValidateFreeformSize(dataLen int) bool {
	return validateBase64Len(dataLen, 1<<21)
}
