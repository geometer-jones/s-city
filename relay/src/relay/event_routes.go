package relay

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"s-city/src/models"
	"s-city/src/services"
	"s-city/src/storage"
)

type EventRoutes struct {
	IngestService *services.EventIngestService
	QueryService  *services.EventQueryService
	DeleteService *services.EventDeleteService
	Logger        *slog.Logger
}

func RegisterEventRoutes(mux *http.ServeMux, routes EventRoutes) {
	mux.HandleFunc("/events", routes.handleEvents)
	mux.HandleFunc("/events/", routes.handleEventSubroutes)
}

func (r EventRoutes) handleEvents(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		var event models.Event
		if err := json.NewDecoder(req.Body).Decode(&event); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event payload"})
			return
		}
		if err := r.IngestService.Ingest(req.Context(), event); err != nil {
			r.Logger.Warn("reject event", "error", err)
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "rate limit") {
				status = http.StatusTooManyRequests
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})

	case http.MethodGet:
		filter, err := parseEventFilter(req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		events, err := r.QueryService.QueryEvents(req.Context(), filter)
		if err != nil {
			r.Logger.Error("query events failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		writeJSON(w, http.StatusOK, events)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (r EventRoutes) handleEventSubroutes(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/events/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "delete" || strings.TrimSpace(parts[0]) == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	eventID := parts[0]
	var deleteReq models.DeletedEvent
	if err := json.NewDecoder(req.Body).Decode(&deleteReq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid delete payload"})
		return
	}
	deleteReq.EventID = eventID

	if err := r.DeleteService.DeleteEvent(req.Context(), deleteReq); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "authorized") {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func parseEventFilter(req *http.Request) (storage.EventFilter, error) {
	q := req.URL.Query()
	filter := storage.EventFilter{Author: q.Get("author"), Tag: q.Get("tag")}

	if kindRaw := q.Get("kind"); kindRaw != "" {
		kind, err := strconv.Atoi(kindRaw)
		if err != nil {
			return storage.EventFilter{}, err
		}
		filter.Kind = &kind
	}
	if sinceRaw := q.Get("since"); sinceRaw != "" {
		since, err := strconv.ParseInt(sinceRaw, 10, 64)
		if err != nil {
			return storage.EventFilter{}, err
		}
		filter.Since = &since
	}
	if untilRaw := q.Get("until"); untilRaw != "" {
		until, err := strconv.ParseInt(untilRaw, 10, 64)
		if err != nil {
			return storage.EventFilter{}, err
		}
		filter.Until = &until
	}
	if limitRaw := q.Get("limit"); limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil {
			return storage.EventFilter{}, err
		}
		filter.Limit = limit
	}

	return filter, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
