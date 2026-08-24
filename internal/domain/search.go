package domain

import (
	"sort"
	"strings"
)

type SortField string

const (
	SortCreated SortField = "created_at"
	SortExpiry  SortField = "expires_at"
	SortCode    SortField = "task_code"
)

type SearchRequest struct {
	Filter RecoveryJobFilter
	Sort   SortField
	Desc   bool
	Offset int
	Limit  int
}

func (r SearchRequest) Normalize() SearchRequest {
	r.Filter = r.Filter.Normalize()
	if r.Sort == "" {
		r.Sort = SortCreated
	}
	if r.Offset < 0 {
		r.Offset = 0
	}
	if r.Limit < 1 || r.Limit > 100 {
		r.Limit = 50
	}
	return r
}

func SearchRecoveryJobs(items []RecoveryJob, request SearchRequest) []RecoveryJob {
	request = request.Normalize()
	filtered := make([]RecoveryJob, 0, len(items))
	for _, item := range items {
		if request.Filter.Matches(item) {
			filtered = append(filtered, item)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left, right := i, j
		if request.Desc {
			left, right = j, i
		}
		switch request.Sort {
		case SortExpiry:
			return filtered[left].ExpiresAt.Before(filtered[right].ExpiresAt)
		case SortCode:
			return strings.ToLower(filtered[left].TaskCode) < strings.ToLower(filtered[right].TaskCode)
		default:
			return filtered[left].ID < filtered[right].ID
		}
	})
	start := request.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + request.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return append([]RecoveryJob(nil), filtered[start:end]...)
}
