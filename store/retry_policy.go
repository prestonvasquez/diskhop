package store

// RetryPolicy defines the retry policy for the store. Retries are only applied
// to store-specified transient errors. For example, if a server is not
// available at the time of a push.
type RetryPolicy struct {
	MaxRetries int // The maximum number of retries to attempt.
}
