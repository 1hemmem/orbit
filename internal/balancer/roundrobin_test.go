package balancer

import (
	"testing"

	"orbit/internal/backend"
)

func TestNextRotates(t *testing.T) {
	pool := backend.NewPool("a", "b", "c")
	rr := New(pool)
	got := []string{rr.Next(), rr.Next(), rr.Next(), rr.Next()}
	want := []string{"a", "b", "c", "a"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}

func TestNextSkipsUnhealthy(t *testing.T) {
	pool := backend.NewPool("a", "b", "c")
	pool.All()[1].SetHealthy(false)
	rr := New(pool)
	got := []string{rr.Next(), rr.Next(), rr.Next(), rr.Next()}
	want := []string{"a", "c", "a", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("want %v, got %v", want, got)
		}
	}
}
