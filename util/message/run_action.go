package message

// RunActionRequest запрос к machine_node для запуска действия.
type RunActionRequest struct {
	Name     string   `json:"name"`
	Action   string   `json:"action"`
	Port     int      `json:"port"`
	Replicas int      `json:"replicas"`
	In       []string `json:"in"`
	Out      []string `json:"out"`
}
