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
