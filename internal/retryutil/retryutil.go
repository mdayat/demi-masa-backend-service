package retryutil

import "github.com/avast/retry-go/v4"

func RetryWithData[T any](f func() (T, error)) (T, error) {
	return retry.DoWithData(f, retry.Attempts(3), retry.LastErrorOnly(true))
}

func RetryWithoutData(f func() error) error {
	return retry.Do(f, retry.Attempts(3), retry.LastErrorOnly(true))
}
