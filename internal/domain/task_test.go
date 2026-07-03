package domain_test

import (
	"testing"

	"github.com/rodrigocavalhero/nanojira/internal/domain"
)

func TestForwardTransitions(t *testing.T) {
	cases := []struct {
		from, to domain.TaskStatus
		want     bool
	}{
		{domain.StatusTodo, domain.StatusDoing, true},
		{domain.StatusDoing, domain.StatusTesting, true},
		{domain.StatusTesting, domain.StatusDone, true},
		{domain.StatusDoing, domain.StatusOnHold, true},
		{domain.StatusOnHold, domain.StatusDoing, true},
		{domain.StatusTodo, domain.StatusDone, false},
		{domain.StatusTesting, domain.StatusDoing, false},
	}

	for _, tc := range cases {
		got := domain.IsForwardTransition(tc.from, tc.to)
		if got != tc.want {
			t.Errorf("forward %s -> %s: got %v want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestBackwardTransitions(t *testing.T) {
	cases := []struct {
		from, to domain.TaskStatus
		want     bool
	}{
		{domain.StatusTesting, domain.StatusDoing, true},
		{domain.StatusDoing, domain.StatusTodo, true},
		{domain.StatusDone, domain.StatusTesting, true},
		{domain.StatusDoing, domain.StatusTesting, false},
	}

	for _, tc := range cases {
		got := domain.IsBackwardTransition(tc.from, tc.to)
		if got != tc.want {
			t.Errorf("backward %s -> %s: got %v want %v", tc.from, tc.to, got, tc.want)
		}
	}
}
