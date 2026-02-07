package relay

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"s-city/src/models"
	"s-city/src/services"
	"s-city/src/storage"
)

type GroupRoutes struct {
	Repo              *storage.GroupRepo
	ProjectionService *services.GroupProjectionService
	Logger            *slog.Logger
}

func RegisterGroupRoutes(mux *http.ServeMux, routes GroupRoutes) {
	mux.HandleFunc("/groups", routes.handleGroups)
	mux.HandleFunc("/groups/", routes.handleGroupSubroutes)
}

func (r GroupRoutes) handleGroups(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	filter, err := parseGroupFilter(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	groups, err := r.Repo.ListGroups(req.Context(), filter)
	if err != nil {
		r.Logger.Error("list groups failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (r GroupRoutes) handleGroupSubroutes(w http.ResponseWriter, req *http.Request) {
	parts := splitPath(strings.TrimPrefix(req.URL.Path, "/groups/"))
	if len(parts) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	groupID := parts[0]

	if len(parts) == 1 {
		r.handleGroupByID(w, req, groupID)
		return
	}

	if len(parts) == 2 {
		switch parts[1] {
		case "members":
			r.handleGroupMembers(w, req, groupID)
		case "roles":
			r.handleGroupRoles(w, req, groupID)
		case "bans":
			r.handleGroupBans(w, req, groupID)
		case "invites":
			r.handleGroupInvites(w, req, groupID)
		case "join-requests":
			r.handleJoinRequests(w, req, groupID)
		default:
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		}
		return
	}

	if len(parts) == 4 && parts[1] == "join-requests" && parts[3] == "approve" {
		r.handleApproveJoinRequest(w, req, groupID, parts[2])
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func (r GroupRoutes) handleGroupByID(w http.ResponseWriter, req *http.Request, groupID string) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	group, err := r.Repo.GetGroup(req.Context(), groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		r.Logger.Error("get group failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (r GroupRoutes) handleGroupMembers(w http.ResponseWriter, req *http.Request, groupID string) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	items, err := r.Repo.ListMembers(req.Context(), groupID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (r GroupRoutes) handleGroupRoles(w http.ResponseWriter, req *http.Request, groupID string) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	items, err := r.Repo.ListRoles(req.Context(), groupID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (r GroupRoutes) handleGroupBans(w http.ResponseWriter, req *http.Request, groupID string) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	items, err := r.Repo.ListBans(req.Context(), groupID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (r GroupRoutes) handleGroupInvites(w http.ResponseWriter, req *http.Request, groupID string) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	items, err := r.Repo.ListInvites(req.Context(), groupID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (r GroupRoutes) handleJoinRequests(w http.ResponseWriter, req *http.Request, groupID string) {
	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var joinRequest models.GroupJoinRequest
	if err := json.NewDecoder(req.Body).Decode(&joinRequest); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	joinRequest.GroupID = groupID
	if joinRequest.CreatedAt == 0 {
		joinRequest.CreatedAt = time.Now().Unix()
	}

	if err := r.Repo.UpsertJoinRequest(req.Context(), joinRequest); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (r GroupRoutes) handleApproveJoinRequest(w http.ResponseWriter, req *http.Request, groupID, pubKey string) {
	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	approver := req.Header.Get("X-Pubkey")
	if approver == "" {
		approver = req.URL.Query().Get("approved_by")
	}
	if approver == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "approved_by (or X-Pubkey) is required"})
		return
	}

	if err := r.ProjectionService.ApproveJoinRequest(req.Context(), groupID, pubKey, approver, time.Now().Unix()); err != nil {
		status := http.StatusForbidden
		if !strings.Contains(err.Error(), "authorized") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func parseGroupFilter(req *http.Request) (storage.GroupFilter, error) {
	q := req.URL.Query()
	filter := storage.GroupFilter{GeohashPrefix: q.Get("geohash_prefix")}

	if v := q.Get("is_private"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return storage.GroupFilter{}, err
		}
		filter.IsPrivate = &parsed
	}
	if v := q.Get("is_vetted"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return storage.GroupFilter{}, err
		}
		filter.IsVetted = &parsed
	}
	if v := q.Get("updated_since"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return storage.GroupFilter{}, err
		}
		filter.UpdatedSince = &parsed
	}
	if v := q.Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return storage.GroupFilter{}, err
		}
		filter.Limit = parsed
	}
	return filter, nil
}

func splitPath(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
