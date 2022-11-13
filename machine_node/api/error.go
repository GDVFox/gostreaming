package api

// Возможные коды ошибок.
var (
	BadUnmarshalRequestErrorCode = "bad_unmarshal"
	NoActionErrorCode            = "action_not_found"
	ETCDErrorCode                = "etcd_error"
	InternalError                = "internal_error"
	BadTelemetry                 = "bad_telemetry"
)
