package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FilterParams represents filtering parameters
type FilterParams struct {
	Filters map[string]string `json:"filters"`
	Sort    SortParams        `json:"sort"`
	Page    int               `json:"page"`
	Limit   int               `json:"limit"`
	Search  string            `json:"search"`
}

// SortParams represents sorting parameters
type SortParams struct {
	Field string `json:"field"`
	Order string `json:"order"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// ParseQueryParams extracts standardized query parameters from Gin context
func ParseQueryParams(c *gin.Context) FilterParams {
	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	// Parse search
	search := c.Query("search")

	// Parse filters - format: filters[field_name]=value
	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "filters[") && strings.HasSuffix(key, "]") {
			fieldName := key[8 : len(key)-1] // Extract field name from filters[field_name]
			if len(values) > 0 && values[0] != "" {
				filters[fieldName] = values[0]
			}
		}
	}

	// Parse sorting - format: sort[field]=field_name&sort[order]=asc|desc
	sortField := c.Query("sort[field]")
	sortOrder := c.Query("sort[order]")

	// Default sorting
	if sortField == "" {
		sortField = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// Validate sort order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	return FilterParams{
		Filters: filters,
		Sort: SortParams{
			Field: sortField,
			Order: sortOrder,
		},
		Page:   page,
		Limit:  limit,
		Search: search,
	}
}

// ApplyFilters applies filters to a GORM query
func ApplyFilters(query *gorm.DB, filters map[string]string, allowedFields map[string]string) *gorm.DB {
	for field, value := range filters {
		if dbField, allowed := allowedFields[field]; allowed && value != "" {
			query = query.Where(fmt.Sprintf("%s = ?", dbField), value)
		}
	}
	return query
}

// ApplySearch applies search to specified fields
func ApplySearch(query *gorm.DB, search string, searchFields []string) *gorm.DB {
	if search == "" || len(searchFields) == 0 {
		return query
	}

	conditions := make([]string, len(searchFields))
	args := make([]interface{}, len(searchFields))

	for i, field := range searchFields {
		conditions[i] = fmt.Sprintf("%s ILIKE ?", field)
		args[i] = "%" + search + "%"
	}

	whereClause := strings.Join(conditions, " OR ")
	return query.Where(whereClause, args...)
}

// ApplySort applies sorting to a GORM query
func ApplySort(query *gorm.DB, sort SortParams, allowedSortFields map[string]string) *gorm.DB {
	if dbField, allowed := allowedSortFields[sort.Field]; allowed {
		orderClause := fmt.Sprintf("%s %s", dbField, strings.ToUpper(sort.Order))
		return query.Order(orderClause)
	}

	// Default sorting if field not allowed
	return query.Order("created_at DESC")
}

// ApplyPagination applies pagination to a GORM query
func ApplyPagination(query *gorm.DB, page, limit int) *gorm.DB {
	offset := (page - 1) * limit
	return query.Offset(offset).Limit(limit)
}

// BuildPaginationResponse creates pagination metadata
func BuildPaginationResponse(page, limit int, total int64) PaginationResponse {
	totalPages := (total + int64(limit) - 1) / int64(limit)
	hasNext := page < int(totalPages)
	hasPrev := page > 1

	return PaginationResponse{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}
}
