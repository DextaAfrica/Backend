package handlers

import (
	"net/http"
	"strconv"
)

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func paginationMeta(page, perPage, total int) (int, int, int) {
	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}
	return page, perPage, totalPages
}
