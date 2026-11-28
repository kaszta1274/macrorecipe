package handler

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/kaszta1274/macrorecipe/internal/api/request"
)

func TestNewPage(t *testing.T) {
	double := func(n int) int { return n * 2 }

	cases := []struct {
		name        string
		rows        []int
		pagination  request.Pagination
		wantItems   []int
		wantHasMore bool
	}{
		{
			name:        "fewer than limit",
			rows:        []int{1, 2},
			pagination:  request.Pagination{Page: 1, Limit: 5},
			wantItems:   []int{2, 4},
			wantHasMore: false,
		},
		{
			name:        "exact limit",
			rows:        []int{1, 2},
			pagination:  request.Pagination{Page: 2, Limit: 2},
			wantItems:   []int{2, 4},
			wantHasMore: false,
		},
		{
			name:        "extra row sets has_more and trims",
			rows:        []int{1, 2, 3},
			pagination:  request.Pagination{Page: 1, Limit: 2},
			wantItems:   []int{2, 4},
			wantHasMore: true,
		},
		{
			name:        "nil rows produces empty non-nil items",
			rows:        nil,
			pagination:  request.Pagination{Page: 1, Limit: 10},
			wantItems:   []int{},
			wantHasMore: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := newPage(tc.rows, tc.pagination, double)
			if got.Items == nil {
				t.Fatal("Items is nil, want non-nil slice")
			}
			if !slices.Equal(got.Items, tc.wantItems) {
				t.Fatalf("Items = %v, want %v", got.Items, tc.wantItems)
			}
			if got.Page != tc.pagination.Page || got.Limit != tc.pagination.Limit {
				t.Fatalf("(Page, Limit) = (%d, %d), want (%d, %d)", got.Page, got.Limit, tc.pagination.Page, tc.pagination.Limit)
			}
			if got.HasMore != tc.wantHasMore {
				t.Fatalf("HasMore = %v, want %v", got.HasMore, tc.wantHasMore)
			}
		})
	}

	raw, err := json.Marshal(newPage(nil, request.Pagination{Page: 1, Limit: 10}, double))
	if err != nil {
		t.Fatalf("marshal empty page: %v", err)
	}
	if !strings.Contains(string(raw), `"items":[]`) {
		t.Fatalf("empty page JSON = %s, want \"items\":[]", raw)
	}
}
