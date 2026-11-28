package request

import (
	"net/http"
	"strconv"
)

type Pagination struct {
	Page   int
	Limit  int
	Offset int
	Query  string
}

func ParsePagination(r *http.Request) Pagination {
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

	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
		Query:  r.URL.Query().Get("query"),
	}
}
