package model

// Pagination adalah metadata yang menyertai setiap response list.
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewPagination menghitung total halaman dari total baris.
func NewPagination(page, limit, total int) Pagination {
	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

// DashboardStats merangkum pekerjaan user untuk halaman dashboard.
type DashboardStats struct {
	TotalTasks int            `json:"total_tasks"`
	ByStatus   map[string]int `json:"by_status"`
	ByPriority map[string]int `json:"by_priority"`
	Overdue    int            `json:"overdue"`
	DueToday   int            `json:"due_today"`
	Projects   int            `json:"projects"`
}
