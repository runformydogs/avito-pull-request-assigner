package apperrors

import "errors"

var (
	ErrPRExists             = errors.New("PR already exists")
	ErrPRNotFound           = errors.New("PR not found")
	ErrPRAuthorNotFound     = errors.New("PR author not found")
	ErrPRTeamNotFound       = errors.New("PR author team not found")
	ErrPRAlreadyMerged      = errors.New("PR already merged")
	ErrReviewerNotAssigned  = errors.New("reviewer is not assigned to this PR")
	ErrNoReviewerCandidates = errors.New("no active replacement candidate in team")
	ErrPRIDRequired         = errors.New("pull request id is required")
	ErrPRNameRequired       = errors.New("pull request name is required")
	ErrAuthorRequired       = errors.New("author id is required")
	ErrOldReviewerRequired  = errors.New("old reviewer id is required")
)
