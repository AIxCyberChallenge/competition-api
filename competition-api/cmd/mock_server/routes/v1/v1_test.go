package v1

import (
	"encoding/base64"

	"github.com/labstack/echo/v4"
)

// Does a request and forwards errors to the error handler like the normal execution path
func doRequest(e *echo.Echo, c echo.Context, handler echo.HandlerFunc) {
	err := handler(c)
	if err != nil {
		e.HTTPErrorHandler(err, c)
	}
}

func base64String(length int) string {
	arr := make([]byte, length)
	for i := range arr {
		arr[i] = 'a'
	}
	return base64.StdEncoding.EncodeToString(arr)
}
