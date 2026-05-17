package util

import (
	"strings"
	"building-services/analytics-service/internal/repository"
)

func AnalyticsFilterFrom(departmentID, projectID, projectIDsFromGateway, fromDate, toDate string) repository.AnalyticsFilter {
	return repository.NewAnalyticsFilter(
		departmentID,
		projectID,
		SplitProjectIDs(projectIDsFromGateway),
		fromDate,
		toDate,
	)
}
func SplitProjectIDs(projectIDsFromGateway string) []string {
	s := strings.TrimSpace(projectIDsFromGateway)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
