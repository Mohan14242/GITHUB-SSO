package cicd

// Event types
const (
	EventStageUpdated  = "stage_updated"
	EventRunUpdated    = "run_updated"
	EventRunCompleted  = "run_completed"
)

type StagePayload struct {
	ID          int64  `json:"id"`
	RunID       int64  `json:"runId"`
	StageName   string `json:"stageName"`
	StageOrder  int    `json:"stageOrder"`
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
	Logs        string `json:"logs"`
}

type RunPayload struct {
	ID          int64  `json:"id"`
	ServiceName string `json:"serviceName"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
}

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}