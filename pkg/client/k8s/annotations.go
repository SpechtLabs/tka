package k8s

// Annotation keys used on TKA resources to track sign-in metadata.
const (
	// LastAttemptedSignIn stores the timestamp of the last sign-in attempt.
	LastAttemptedSignIn = "tka.specht-labs.de/last-attempted-sign-in"
	// SignInValidUntil stores the expiration timestamp of the current sign-in.
	SignInValidUntil = "tka.specht-labs.de/sign-in-valid-until"
)
