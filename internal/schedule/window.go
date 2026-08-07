// Package schedule evaluates KICK-native execution windows without a GitOps provider.
package schedule

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// Window is a single native allow/deny window.
type Window struct {
	Allow    bool
	Schedule string
	Duration string
	TimeZone string
}

// Decision is the outcome of evaluating a set of native windows.
type Decision struct {
	Allowed   bool
	RequeueAt *time.Time
}

// Evaluate applies allow/deny precedence: an active deny blocks; if any allow
// window exists and none is active the action is blocked; otherwise allowed.
func Evaluate(now time.Time, windows []Window) (Decision, error) {
	if len(windows) == 0 {
		return Decision{Allowed: true}, nil
	}

	denyActive := false
	hasAllow := false
	allowActive := false
	var next *time.Time

	for _, w := range windows {
		active, windowNext, err := windowActive(now, w)
		if err != nil {
			return Decision{}, err
		}
		next = minTime(next, windowNext)
		if w.Allow {
			hasAllow = true
			if active {
				allowActive = true
			}
		} else if active {
			denyActive = true
		}
	}

	if denyActive {
		return Decision{Allowed: false, RequeueAt: next}, nil
	}
	if hasAllow && !allowActive {
		return Decision{Allowed: false, RequeueAt: next}, nil
	}
	return Decision{Allowed: true, RequeueAt: next}, nil
}

func windowActive(now time.Time, w Window) (bool, *time.Time, error) {
	if w.Schedule == "" {
		return false, nil, fmt.Errorf("missing window schedule")
	}
	dur, err := time.ParseDuration(w.Duration)
	if err != nil {
		return false, nil, fmt.Errorf("parse duration: %w", err)
	}
	location := time.UTC
	if w.TimeZone != "" {
		loc, tzErr := time.LoadLocation(w.TimeZone)
		if tzErr != nil {
			return false, nil, fmt.Errorf("load timezone: %w", tzErr)
		}
		location = loc
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	spec, err := parser.Parse(w.Schedule)
	if err != nil {
		return false, nil, fmt.Errorf("parse cron schedule: %w", err)
	}

	localNow := now.In(location)
	candidateStart := spec.Next(localNow.Add(-dur))
	active := !candidateStart.After(localNow)
	nextStart := spec.Next(localNow)
	nextEnd := candidateStart.Add(dur)

	next := nextStart.UTC()
	if active && nextEnd.Before(next) {
		t := nextEnd.UTC()
		next = t
	}
	return active, &next, nil
}

func minTime(a, b *time.Time) *time.Time {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if b.Before(*a) {
		return b
	}
	return a
}
