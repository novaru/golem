package balancer

import "testing"

func TestBalancerFactory(t *testing.T) {
	backends := []*Backend{
		NewBackend("http://a", 1),
		NewBackend("http://b", 1),
	}
	b, err := NewBalancer("roundrobin", backends)
	if err != nil || b == nil {
		t.Errorf("expected roundrobin balancer, got err=%v", err)
	}
	b, err = NewBalancer("unknown", backends)
	if err == nil {
		t.Error("expected error for unknown method")
	}
}
