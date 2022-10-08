package common

// Возможные коды ошибок.
var (
	BadUnmarshalRequestErrorCode = "bad_unmarshal"
	BadSchemeErrorCode           = "bad_scheme"
	BadActionErrorCode           = "bad_action"
	BadNameErrorCode             = "bad_name"
	NameNotFoundErrorCode        = "name_not_found"
	NameAlreadyExistsErrorCode   = "name_already_exists"
	ETCDErrorCode                = "etcd_error"
	MachineErrorCode             = "machine_error"
)
