package ratelimitlx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	applicationJson      = "application/json"
	hset                 = "HSET"
	hget                 = "HGET"
	hincrby              = "HINCRBY"
	unableToWriteToCache = "unable to write jwt to cache"
	ratelimits           = "rate_limits"
	colonDelimiter       = ":"
)

var (
	localCacheAddress = os.Getenv("LOCAL_CACHE_ADDRESS")

	errInstructionsAreNil       = errors.New("instructions are nil")
	errRequestToCacheFailed     = errors.New("request to local cache failed")
	errInstructionsFailed       = errors.New("instructions failed")
	errCurrIntervalIsAboveLimit = errors.New("current interval is above limit")
)

func getCacheSetID(categories ...string) string {
	return strings.Join(categories, colonDelimiter)
}

func execInstructionsAndParseInt64(
	instructions *[]interface{},
) (*int64, error) {
	if instructions == nil {
		return nil, errInstructionsAreNil
	}

	bodyBytes := new(bytes.Buffer)
	errJson := json.NewEncoder(bodyBytes).Encode(*instructions)
	if errJson != nil {
		return nil, errJson
	}

	resp, errResp := http.Post(localCacheAddress, applicationJson, bodyBytes)
	if errResp != nil {
		return nil, errResp
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, errRequestToCacheFailed
	}

	var count int64
	errCount := json.NewDecoder(resp.Body).Decode(&count)
	if errCount != nil {
		return nil, errCount
	}
	if &count != nil {
		return &count, nil
	}

	return nil, errInstructionsFailed
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
	interval int64, // by seconds
) (*string, *string) {
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

func Limit(
	serverName string,
	identifier string,
	limit int64,
	intervalWindow int64, // by seconds
	currentTime int64,
) (bool, error) {
	prevIntervalID, currIntervalID := getIntervals(
		serverName,
		intervalWindow,
		currentTime,
	)

	setID := getCacheSetID(serverName, ratelimits)

	incrCurrCount := []interface{}{hincrby, setID, currIntervalID, 1}
	currCount, errCurrCount := execInstructionsAndParseInt64(&incrCurrCount)
	if errCurrCount != nil {
		return false, errCurrCount
	}

	getPrevCount := []interface{}{hget, setID, prevIntervalID}
	prevCount, errPrevCount := execInstructionsAndParseInt64(&getPrevCount)
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

	return false, nil
}
