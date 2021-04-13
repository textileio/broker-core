package auth

import "context"

// Authorizer provides authorization resolving for upload requests.
type Authorizer interface {
	// IsAuthorized indicates if the identity is authorized
	// to call the upload endpoint. If 'false' is returned, it also
	// returns a string with an explanation of why that's the case.
	IsAuthorized(ctx context.Context, identity string) (bool, string, error)
}
