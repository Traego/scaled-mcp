package protocol

func IsOnewayMethod(method string) bool {
	return method == "notifications/initialized"
}
