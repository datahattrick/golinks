package db

import "errors"

// Domain-level database error sentinels.
var (
	// Link errors
	ErrLinkNotFound     = errors.New("link not found")
	ErrDuplicateKeyword = errors.New("keyword already exists")

	// User errors
	ErrUserNotFound = errors.New("user not found")

	// Organisation errors
	ErrOrgNotFound = errors.New("organization not found")

	// User link errors
	ErrUserLinkNotFound = errors.New("user link not found")

	// Shared link errors
	ErrShareLimitReached     = errors.New("you have reached the maximum number of pending outgoing shares")
	ErrRecipientLimitReached = errors.New("recipient has reached the maximum number of pending incoming shares")
	ErrDuplicateShare        = errors.New("you have already shared this keyword with this user")
	ErrSharedLinkNotFound    = errors.New("shared link not found")

	// Edit request errors
	ErrEditRequestNotFound  = errors.New("edit request not found")
	ErrPendingRequestLimit  = errors.New("you have reached the maximum number of pending requests (5)")
	ErrDuplicateEditRequest = errors.New("you already have a pending edit request for this link")

	// Fallback redirect errors
	ErrFallbackRedirectNotFound = errors.New("fallback redirect not found")
)
