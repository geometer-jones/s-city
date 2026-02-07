package relay

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEventRoutesHandlerGuards(t *testing.T) {
	routes := EventRoutes{}

	t.Run("handleEvents rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/events", nil)
		rec := httptest.NewRecorder()
		routes.handleEvents(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleEvents rejects malformed post payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString("{"))
		rec := httptest.NewRecorder()
		routes.handleEvents(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("handleEventSubroutes rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/events/evt-1/delete", nil)
		rec := httptest.NewRecorder()
		routes.handleEventSubroutes(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleEventSubroutes rejects unknown subroute", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/events/evt-1/unknown", nil)
		rec := httptest.NewRecorder()
		routes.handleEventSubroutes(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("handleEventSubroutes rejects malformed delete payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/events/evt-1/delete", bytes.NewBufferString("{"))
		rec := httptest.NewRecorder()
		routes.handleEventSubroutes(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestGroupRoutesHandlerGuards(t *testing.T) {
	routes := GroupRoutes{}

	t.Run("handleGroups rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups", nil)
		rec := httptest.NewRecorder()
		routes.handleGroups(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleGroups rejects invalid filter query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/groups?limit=bad", nil)
		rec := httptest.NewRecorder()
		routes.handleGroups(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("handleGroupSubroutes rejects empty path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/groups/", nil)
		rec := httptest.NewRecorder()
		routes.handleGroupSubroutes(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("handleGroupSubroutes rejects unknown subroute", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/groups/group-1/unknown", nil)
		rec := httptest.NewRecorder()
		routes.handleGroupSubroutes(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("handleGroupSubroutes rejects unknown deep path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/groups/group-1/a/b", nil)
		rec := httptest.NewRecorder()
		routes.handleGroupSubroutes(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("handleGroupByID rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups/group-1", nil)
		rec := httptest.NewRecorder()
		routes.handleGroupByID(rec, req, "group-1")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleGroupMembers rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups/group-1/members", nil)
		rec := httptest.NewRecorder()
		routes.handleGroupMembers(rec, req, "group-1")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleGroupRoles rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups/group-1/roles", nil)
		rec := httptest.NewRecorder()
		routes.handleGroupRoles(rec, req, "group-1")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleGroupBans rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups/group-1/bans", nil)
		rec := httptest.NewRecorder()
		routes.handleGroupBans(rec, req, "group-1")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleGroupInvites rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups/group-1/invites", nil)
		rec := httptest.NewRecorder()
		routes.handleGroupInvites(rec, req, "group-1")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleJoinRequests rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/groups/group-1/join-requests", nil)
		rec := httptest.NewRecorder()
		routes.handleJoinRequests(rec, req, "group-1")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleJoinRequests rejects malformed payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups/group-1/join-requests", bytes.NewBufferString("{"))
		rec := httptest.NewRecorder()
		routes.handleJoinRequests(rec, req, "group-1")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("handleApproveJoinRequest rejects unsupported method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/groups/group-1/join-requests/user-1/approve", nil)
		rec := httptest.NewRecorder()
		routes.handleApproveJoinRequest(rec, req, "group-1", "user-1")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handleApproveJoinRequest requires approver", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/groups/group-1/join-requests/user-1/approve", nil)
		rec := httptest.NewRecorder()
		routes.handleApproveJoinRequest(rec, req, "group-1", "user-1")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
