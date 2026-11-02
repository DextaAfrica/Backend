package service

import "github.com/DextaAfrica/Backend/internal/domain"

const (
	defaultPerPage = 20
	maxPerPage     = 100
)

// NormalizeListParams clamps caller-supplied paging values to sane bounds so
// a malicious or buggy client can't request page=-5 or per_page=100000 and
// force a full table scan.
func NormalizeListParams(page, perPage int, publishedOnly bool) domain.ListParams {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return domain.ListParams{Page: page, PerPage: perPage, PublishedOnly: publishedOnly}
}
