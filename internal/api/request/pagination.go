package request

import (
	"net/http"
	"strconv"
)

func ParsePagination(r *http.Request) (int, int, string) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	query := r.URL.Query().Get("query")

	return limit, offset, query
}
