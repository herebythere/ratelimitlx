package ratelimitlx

import (
	"errors"
	"fmt"
	"strings"

	sclx "github.com/herebythere/supercachelx/v0.1/golang"
)

const (
	expireCache         = "EXPIRE"
	getCache            = "GET"
	incrCache           = "INCR"
	ratelimits          = "rate_limits"
	colonDelimiter      = ":"
	threeHoursInSeconds = 10800
	okCache             = "OK"
)

var (
	errCurrIntervalIsAboveLimit = errors.New("current interval is above limit")
	errExpiryFailed             = errors.New("setting expiry failed")
)

func getCacheSetID(categories ...string) string {
	return strings.Join(categories, colonDelimiter)
}

func minInt(x, y int64) int64 {
	if x < y {
		return x
	}

	return y
}

func slidingWindowLimit(
	limit int64,
	interval int64,
	prevCount int64,
	currCount int64,
	currentTime int64,
) bool {
	if limit <= 0 {
		return false // add error
	}
	if interval <= 0 {
		return false // add error
	}
	if currCount > limit {
		return false
	}

	adjPrevCount := float64(minInt(prevCount, limit))
	adjCurrCount := minInt(currCount, limit)
	intervalValue := float64(currentTime % interval)
	intervalDelta := 1 - (intervalValue / float64(interval))
	windowValue := int64(intervalDelta * adjPrevCount)
	totalCount := windowValue + adjCurrCount

	if totalCount < limit {
		return true
	}

	return false
}

func getIntervals(
	identifier string,
	currentTime int64,
	interval int64,
) (
	*string,
	*string,
) {
	currentInterval := currentTime / interval
	currIntervalID := getCacheSetID(
		identifier,
		fmt.Sprint(currentInterval),
	)

	previousInterval := currentInterval - 1
	prevIntervalID := getCacheSetID(
		identifier,
		fmt.Sprint(previousInterval),
	)

	return &prevIntervalID, &currIntervalID
}

func expireInterval(
	cacheAddress string,
	itervalID string,
) error {
	expCurrent := []interface{}{expireCache, itervalID, threeHoursInSeconds}
	expSuccess, errExp := sclx.ExecInstructionsAndParseString(
		cacheAddress,
		&expCurrent,
	)
	if errExp != nil {
		return errExp
	}
	if (*expSuccess) == okCache {
		return nil
	}

	return errExpiryFailed
}

func Limit(
	cacheAddress string,
	identifier string,
	limit int64,
	intervalWindow int64,
	currentTime int64,
) (
	bool,
	error,
) {
	prevIntervalID, currIntervalID := getIntervals(
		identifier,
		intervalWindow,
		currentTime,
	)

	currSetID := getCacheSetID(
		identifier,
		ratelimits,
		*currIntervalID,
	)

	// increment
	incrCurrCount := []interface{}{incrCache, currSetID}
	currCount, errCurrCount := sclx.ExecInstructionsAndParseInt64(
		cacheAddress,
		&incrCurrCount,
	)
	if errCurrCount != nil {
		return false, errCurrCount
	}
	if *currCount >= limit {
		return false, errCurrIntervalIsAboveLimit
	}

	// set increment expiry
	errExpiry := expireInterval(
		cacheAddress,
		currSetID,
	)
	if errExpiry != nil {
		return false, errExpiry
	}

	// validate bucket if needed
	prevSetID := getCacheSetID(
		identifier,
		ratelimits,
		*prevIntervalID,
	)
	getPrevCount := []interface{}{getCache, prevSetID}
	prevCount, errPrevCount := sclx.ExecInstructionsAndParseInt64(
		cacheAddress,
		&getPrevCount,
	)
	if errPrevCount != nil {
		return false, errPrevCount
	}

	passedWindowLimit := slidingWindowLimit(
		limit,
		intervalWindow,
		*prevCount,
		*currCount,
		currentTime,
	)

	if passedWindowLimit {
		return true, nil
	}

	return false, errCurrIntervalIsAboveLimit
}
