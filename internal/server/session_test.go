package server

import "testing"

func TestSessionStoreLifecycle(t *testing.T) {
	store := newSessionStore()
	store.put("abc", "user-1")
	if !store.valid("abc") {
		t.Fatal("expected session to be valid after put")
	}
	if store.valid("missing") {
		t.Fatal("expected missing session to be invalid")
	}
	store.delete("abc")
	if store.valid("abc") {
		t.Fatal("expected session to be invalid after delete")
	}
}
