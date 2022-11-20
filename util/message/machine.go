package message

// RunActionRequest запрос к machine_node для запуска действия.
type RunActionRequest struct {
	SchemeName string            `json:"scheme_name"`
	ActionName string            `json:"action_name"`
	Action     string            `json:"action"`
	Port       int               `json:"port"`
	In         []string          `json:"in"`
	Out        []string          `json:"out"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env"`
}

// StopActionRequest запрос к machine_node для остановки действия.
type StopActionRequest struct {
	SchemeName string `json:"scheme_name"`
	ActionName string `json:"action_name"`
}

// ChangeOutRequest запрос на замену выходного потока.
type ChangeOutRequest struct {
	SchemeName string `json:"scheme_name"`
	ActionName string `json:"action_name"`
	OldOut     string `json:"old_out"`
	NewOut     string `json:"new_out"`
}

// RuntimeStatus состояние runtime
type RuntimeStatus uint8

const (
	// RuntimeStatusOK runtime работает и отвечает на последний ping.
	RuntimeStatusOK RuntimeStatus = 0
	// RuntimeStatusPending runtime пока считается рабочим, но не ответил на несколько последних ping.
	RuntimeStatusPending RuntimeStatus = 1
)

// RuntimeTelemetry набор информации о рантайме.
type RuntimeTelemetry struct {
	SchemeName   string        `json:"scheme_name"`
	ActionName   string        `json:"action_name"`
	Status       RuntimeStatus `json:"status"`
	OldestOutput uint32        `json:"oldest_output"`
}
