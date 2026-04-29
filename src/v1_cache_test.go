package main

import (
	"errors"
	"testing"
	"time"
)

func TestTTLCacheCachesSuccessfulValues(t *testing.T) {
	oldCache := aggregateCache
	aggregateCache = newTTLCache()
	t.Cleanup(func() { aggregateCache = oldCache })

	loads := 0
	load := func() (int, error) {
		loads++
		return 42, nil
	}
	first, err := cachedValue("cache-hit", time.Minute, load)
	if err != nil || first != 42 {
		t.Fatalf("first load = %d, %v", first, err)
	}
	second, err := cachedValue("cache-hit", time.Minute, load)
	if err != nil || second != 42 {
		t.Fatalf("second load = %d, %v", second, err)
	}
	if loads != 1 {
		t.Fatalf("loader called %d times, want 1", loads)
	}
}

func TestTTLCacheExpiresAndDoesNotCacheErrors(t *testing.T) {
	oldCache := aggregateCache
	aggregateCache = newTTLCache()
	t.Cleanup(func() { aggregateCache = oldCache })

	loads := 0
	_, err := cachedValue("cache-error", time.Minute, func() (int, error) {
		loads++
		return 0, errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected loader error")
	}
	value, err := cachedValue("cache-error", time.Minute, func() (int, error) {
		loads++
		return 7, nil
	})
	if err != nil || value != 7 {
		t.Fatalf("reloaded value = %d, %v", value, err)
	}
	if loads != 2 {
		t.Fatalf("loader called %d times, want 2", loads)
	}

	value, err = cachedValue("cache-expire", time.Nanosecond, func() (int, error) {
		return 1, nil
	})
	if err != nil || value != 1 {
		t.Fatalf("initial expiring value = %d, %v", value, err)
	}
	time.Sleep(time.Millisecond)
	value, err = cachedValue("cache-expire", time.Minute, func() (int, error) {
		return 2, nil
	})
	if err != nil || value != 2 {
		t.Fatalf("expired reload value = %d, %v", value, err)
	}
}
