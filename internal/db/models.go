package db

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type Repo struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	DeployScript string `json:"deploy_script"`
	AddedAt      string `json:"added_at"`
}

type Task struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	RepoID       int64  `json:"repo_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	BranchName   string `json:"branch_name"`
	PRNumber     int    `json:"pr_number"`
	PlanText     string `json:"plan_text"`
	ErrorMessage string `json:"error_message"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type TaskLog struct {
	ID        int64  `json:"id"`
	TaskID    int64  `json:"task_id"`
	Stage     string `json:"stage"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ChatMessage struct {
	ID        int64  `json:"id"`
	TaskID    int64  `json:"task_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}
