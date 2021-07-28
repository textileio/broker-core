package auth

import "context"

// AuthorizedEntity contains identity and origin information form an authorizer entity.
type AuthorizedEntity struct {
	Identity string
	Origin   string
}

// Authorizer provides authorization resolving for upload requests.
type Authorizer interface {
	// IsAuthorized indicates if the token is authorized
	// to call the upload endpoint. If 'false' is returned, it also
	// returns a string with an explanation of why that's the case.
	IsAuthorized(ctx context.Context, token string) (AuthorizedEntity, bool, string, error)
}
