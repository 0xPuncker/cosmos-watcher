package types

// Job represents a scheduled job configuration
type Job struct {
	Name        string `json:"name"`
	Schedule    string `json:"schedule"`
	TaskName    string `json:"task"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// JobConfig represents the job scheduler configuration
type JobConfig struct {
	MaxConcurrent int   `json:"max_concurrent"`
	Predefined    []Job `json:"predefined"`
}
