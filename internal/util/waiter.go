package util

import (
	"errors"
	"time"
)

func WaitUntil[T any](get func() (T, error), condition func(T, error) bool, delay time.Duration, maxWaitTime time.Duration) (T, error) {
	end := time.Now().Add(maxWaitTime)
	for {
		res, err := get()
		if condition(res, err) {
			return res, nil
		}
		if time.Now().After(end) {
			return res, errors.New("waiting condition wasn't reached in time")
		}
		time.Sleep(delay)
	}
}
