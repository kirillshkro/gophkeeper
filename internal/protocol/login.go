// Package-level definitions for the login payload and its validation
// (protocol contract, tests-specs section 3.2).
package protocol

// LoginRequest is the payload of the authentication endpoint.
//
// It carries the account credentials used to log in. Only the two credentials
// are validated here (presence and format), not their correctness — the
// server compares the password with the stored hash in the auth package.
type LoginRequest struct {
	// Login is the username of the account.
	Login string `json:"login"`
	// Password is the account password supplied at login time.
	Password string `json:"password"`
}

// ValidateLogin reports whether req is a well-formed login request.
//
// It returns nil when both the login and the password are present and pass
// the same checks used by ValidateRegister. A nil request, an empty or
// malformed login, or an empty or too long password each yield a FieldError
// wrapping ErrInvalidPayload and naming the offending field.
func ValidateLogin(req *LoginRequest) error {
	if req == nil {
		return fieldError("request", "must not be nil")
	}
	if err := validateLogin(req.Login); err != nil {
		return err
	}
	if err := validatePassword(req.Password); err != nil {
		return err
	}
	return nil
}
