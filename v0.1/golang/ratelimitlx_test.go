package ratelimitlx

import (
	"os"
	"testing"

	sclx "github.com/herebythere/supercachelx/v0.1/golang"
)

const (
	testRateLimiter   = "rate_limiter_interface_test"
	altTestIdentifier = "alt_test_identifier"
	testAddress       = "address_test"
)

var (
	localCacheAddress = os.Getenv("LOCAL_CACHE_ADDRESS")
	// localCacheAddress = "http://10.88.0.1:1234"
)

func TestExecInstructionsAndParseInt64(t *testing.T) {
	instructions := []interface{}{incrCache, testRateLimiter}
	count, errCount := sclx.ExecInstructionsAndParseInt64(
		localCacheAddress,
		&instructions,
	)
	if errCount != nil {
		t.Fail()
		t.Logf(errCount.Error())
	}
	if count == nil {
		t.Fail()
		t.Logf("increment was not successfuul")
	}
	if count != nil && *count < 1 {
		t.Fail()
		t.Logf("increment was less than 1, which means nothing was incrmented")
	}
}

func testMinInt(t *testing.T) {
	minZero := minInt(0, 1)
	if minZero != 0 {
		t.Fail()
		t.Logf("min of (0, 1) should be 0")
	}

	minOne := minInt(1, 0)
	if minOne != 0 {
		t.Fail()
		t.Logf("min of (1, 0) should be 0")
	}

	minNegativeOne := minInt(0, -1)
	if minNegativeOne != -1 {
		t.Fail()
		t.Logf("min of (0, -1) should be -1")
	}

	minNegativeOneFirstArg := minInt(-1, 0)
	if minNegativeOneFirstArg != -1 {
		t.Fail()
		t.Logf("min of (-1, 0) should be -1")
	}

	minBothOne := minInt(1, -1)
	if minBothOne != -1 {
		t.Fail()
		t.Logf("min of (1, -1) should be 0")
	}

	minFirstArgNegativeBothOne := minInt(-1, 1)
	if minFirstArgNegativeBothOne != -1 {
		t.Fail()
		t.Logf("min of (-1, 1) should be 0")
	}
}

func TestSlidingWindowLimitOfFifteenSeconds(t *testing.T) {
	allowdInWindow := slidingWindowLimit(
		10,
		15,
		9,
		3,
		8,
	)

	if !allowdInWindow {
		t.Fail()
		t.Logf("sliding window of fifteen seconds denied a valid increment")
	}

	shouldNotAllow := slidingWindowLimit(
		10,
		15,
		9,
		5,
		2,
	)

	if shouldNotAllow {
		t.Fail()
		t.Logf("sliding window of fifteen seconds should deny the increment")
	}
}

func TestSlidingWindowLimitOfOneSecondsNano(t *testing.T) {
	allowdInWindow := slidingWindowLimit(
		10,
		1000000,
		10,
		2,
		250000,
	)

	if !allowdInWindow {
		t.Fail()
		t.Logf("sliding window of one second denied a valid increment")
	}

	shouldNotAllow := slidingWindowLimit(
		10,
		1000000,
		12,
		9,
		1750000,
	)

	if shouldNotAllow {
		t.Fail()
		t.Logf("sliding window of one second should deny the increment")
	}
}

func TestSlidingWindowLimitOfOneSecondLimitTwentyFive(t *testing.T) {
	allowdInWindow := slidingWindowLimit(
		25,
		250000,
		32,
		4,
		1320000,
	)

	if !allowdInWindow {
		t.Fail()
		t.Logf("sliding window of one second denied a valid increment")
	}

	shouldNotAllow := slidingWindowLimit(
		25,
		1000000,
		48,
		19,
		1750000,
	)

	if shouldNotAllow {
		t.Fail()
		t.Logf("sliding window of one second should deny the increment")
	}
}

func TestLimitOfFifteenSeconds(t *testing.T) {
	testLimit := int64(10)
	testLimitInt := int(testLimit + 1)

	var errHasBeenLimited error
	var hasBeenLimited bool

	iteration := 0
	for iteration < testLimitInt {
		passedLimit, errLimited := Limit(
			localCacheAddress,
			testRateLimiter,
			testLimit,
			15,
			2,
		)

		hasBeenLimited = !passedLimit

		iteration += 1
		if errLimited != nil {
			errHasBeenLimited = errLimited
			break
		}
	}

	if errHasBeenLimited == nil {
		t.Fail()
		t.Logf(errHasBeenLimited.Error())
	}
	if !hasBeenLimited {
		t.Fail()
		t.Logf("sliding window of fifteen seconds should deny 11 increments in 15 seconds with a limit of 10")
	}
}
