package argocd

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/corewire/kick/internal/gitops"
	"github.com/robfig/cron/v3"
)

type appWindowContext struct {
	Name              string
	DestinationNS     string
	DestinationName   string
	DestinationServer string
}

type windowEvalResult struct {
	Allowed   bool
	RequeueAt *time.Time
	Reason    gitops.GateReason
}

func evaluateSyncWindows(now time.Time, ctx appWindowContext, windows []map[string]interface{}) (windowEvalResult, error) {
	if len(windows) == 0 {
		return windowEvalResult{Allowed: true, Reason: gitops.GateAllowed}, nil
	}

	denyActive := false
	hasAllow := false
	allowActive := false
	var next *time.Time

	for _, window := range windows {
		match := matchesWindow(ctx, window)
		if !match {
			continue
		}
		kind := strings.ToLower(stringValue(window, "kind"))
		if kind == "allow" {
			hasAllow = true
		}

		active, windowNext, err := isWindowActive(now, window)
		if err != nil {
			return windowEvalResult{}, err
		}
		next = minTime(next, windowNext)

		if kind == "deny" && active {
			denyActive = true
		}
		if kind == "allow" && active {
			allowActive = true
		}
	}

	if denyActive {
		return windowEvalResult{Allowed: false, Reason: gitops.GateOutsideSchedule, RequeueAt: next}, nil
	}
	if hasAllow && !allowActive {
		return windowEvalResult{Allowed: false, Reason: gitops.GateOutsideSchedule, RequeueAt: next}, nil
	}
	return windowEvalResult{Allowed: true, Reason: gitops.GateAllowed, RequeueAt: next}, nil
}

func matchesWindow(ctx appWindowContext, window map[string]interface{}) bool {
	appMatches := listMatches(ctx.Name, listValue(window, "applications"))
	nsMatches := listMatches(ctx.DestinationNS, listValue(window, "namespaces"))
	clusterMatches := listMatchesAny([]string{ctx.DestinationName, ctx.DestinationServer}, listValue(window, "clusters"))

	useAnd := boolValue(window, "useAndOperator")
	if useAnd {
		active := []bool{}
		if hasSelector(window, "applications") {
			active = append(active, appMatches)
		}
		if hasSelector(window, "namespaces") {
			active = append(active, nsMatches)
		}
		if hasSelector(window, "clusters") {
			active = append(active, clusterMatches)
		}
		if len(active) == 0 {
			return false
		}
		for _, ok := range active {
			if !ok {
				return false
			}
		}
		return true
	}
	if hasSelector(window, "applications") && appMatches {
		return true
	}
	if hasSelector(window, "namespaces") && nsMatches {
		return true
	}
	if hasSelector(window, "clusters") && clusterMatches {
		return true
	}
	return false
}

func isWindowActive(now time.Time, window map[string]interface{}) (bool, *time.Time, error) {
	schedule := stringValue(window, "schedule")
	if schedule == "" {
		return false, nil, fmt.Errorf("missing window schedule")
	}
	dur, err := time.ParseDuration(stringValue(window, "duration"))
	if err != nil {
		return false, nil, fmt.Errorf("parse duration: %w", err)
	}
	location := time.UTC
	if tz := stringValue(window, "timeZone"); tz != "" {
		loc, tzErr := time.LoadLocation(tz)
		if tzErr != nil {
			return false, nil, fmt.Errorf("load timezone: %w", tzErr)
		}
		location = loc
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	spec, err := parser.Parse(schedule)
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

func listMatches(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if ok, _ := path.Match(pattern, value); ok {
			return true
		}
	}
	return false
}

func listMatchesAny(values []string, patterns []string) bool {
	for _, v := range values {
		if v == "" {
			continue
		}
		if listMatches(v, patterns) {
			return true
		}
	}
	return false
}

func listValue(window map[string]interface{}, key string) []string {
	raw, ok := window[key]
	if !ok {
		return nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func stringValue(window map[string]interface{}, key string) string {
	if value, ok := window[key].(string); ok {
		return value
	}
	return ""
}

func boolValue(window map[string]interface{}, key string) bool {
	value, _ := window[key].(bool)
	return value
}

func hasSelector(window map[string]interface{}, key string) bool {
	items := listValue(window, key)
	return len(items) > 0
}

func minTime(current *time.Time, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.Before(*current) {
		t := *candidate
		return &t
	}
	return current
}
