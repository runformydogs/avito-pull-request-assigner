package models

type PRStats struct {
	TotalPRs          int     `json:"total_prs"`
	OpenPRs           int     `json:"open_prs"`
	MergedPRs         int     `json:"merged_prs"`
	AvgReviewersPerPR float64 `json:"avg_reviewers_per_pr"`
}
