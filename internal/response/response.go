package response

import (
	"encoding/json"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

func Paginated(w http.ResponseWriter, status int, data any, page, limit, total int) {
	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	JSON(w, status, map[string]any{
		"data": data,
		"pagination": map[string]int{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}
