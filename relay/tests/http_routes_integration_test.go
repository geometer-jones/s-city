package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"s-city/src/lib"
	"s-city/src/models"
	relayhttp "s-city/src/relay"
	"s-city/src/services"
	"s-city/src/storage"
)

func TestEventRoutesHTTP(t *testing.T) {
	pool := openIntegrationPool(t)

	tagsRepo := storage.NewEventTagsRepo()
	eventsRepo := storage.NewEventsRepo(pool, tagsRepo)
	groupRepo := storage.NewGroupRepo(pool)
	metrics := lib.NewMetrics()

	relayPriv, relayPub := generateKeypair(t)
	validator := services.NewValidator(5 * time.Minute)
	abuse := services.NewAbuseControls(100, 600, 0)
	vetting := services.NewGroupVettingService(groupRepo)
	projection := services.NewGroupProjectionService(groupRepo, eventsRepo, relayPub, relayPriv, vetting, metrics)
	ingest := services.NewEventIngestService(eventsRepo, validator, abuse, projection, metrics, relayPub)
	query := services.NewEventQueryService(eventsRepo)
	del := services.NewEventDeleteService(eventsRepo, projection, metrics)

	mux := http.NewServeMux()
	relayhttp.RegisterEventRoutes(mux, relayhttp.EventRoutes{
		IngestService: ingest,
		QueryService:  query,
		DeleteService: del,
		Logger:        lib.NewLogger("ERROR"),
	})

	invalidReq := httptest.NewRequest(http.MethodPost, "/events", bytes.NewBufferString("{"))
	invalidRec := httptest.NewRecorder()
	mux.ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid event payload status = %d, want %d", invalidRec.Code, http.StatusBadRequest)
	}

	priv, pub := generateKeypair(t)
	event := signedModelEvent(t, priv, nowUnix(), 1, [][]string{{"t", "nostr"}}, "hello")
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	postReq := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("POST /events status = %d, want %d body=%s", postRec.Code, http.StatusAccepted, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/events?author="+pub, nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /events status = %d, want %d body=%s", getRec.Code, http.StatusOK, getRec.Body.String())
	}
	var got []models.Event
	if err := json.Unmarshal(getRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode /events response: %v", err)
	}
	if len(got) != 1 || got[0].ID != event.ID {
		t.Fatalf("unexpected /events response: %v", got)
	}

	unauthorizedDeleteBody, _ := json.Marshal(models.DeletedEvent{DeletedBy: "mallory", DeletedAt: nowUnix()})
	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/events/"+event.ID+"/delete", bytes.NewReader(unauthorizedDeleteBody))
	unauthorizedRec := httptest.NewRecorder()
	mux.ServeHTTP(unauthorizedRec, unauthorizedReq)
	if unauthorizedRec.Code != http.StatusForbidden {
		t.Fatalf("unauthorized delete status = %d, want %d body=%s", unauthorizedRec.Code, http.StatusForbidden, unauthorizedRec.Body.String())
	}

	authorizedDeleteBody, _ := json.Marshal(models.DeletedEvent{DeletedBy: pub, DeletedAt: nowUnix(), Reason: "cleanup"})
	authorizedReq := httptest.NewRequest(http.MethodPost, "/events/"+event.ID+"/delete", bytes.NewReader(authorizedDeleteBody))
	authorizedRec := httptest.NewRecorder()
	mux.ServeHTTP(authorizedRec, authorizedReq)
	if authorizedRec.Code != http.StatusAccepted {
		t.Fatalf("authorized delete status = %d, want %d body=%s", authorizedRec.Code, http.StatusAccepted, authorizedRec.Body.String())
	}

	afterDeleteReq := httptest.NewRequest(http.MethodGet, "/events?author="+pub, nil)
	afterDeleteRec := httptest.NewRecorder()
	mux.ServeHTTP(afterDeleteRec, afterDeleteReq)
	if afterDeleteRec.Code != http.StatusOK {
		t.Fatalf("GET /events after delete status = %d, want %d", afterDeleteRec.Code, http.StatusOK)
	}
	got = nil
	if err := json.Unmarshal(afterDeleteRec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode /events after delete response: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no visible events after delete, got %v", got)
	}

	methodReq := httptest.NewRequest(http.MethodPut, "/events", nil)
	methodRec := httptest.NewRecorder()
	mux.ServeHTTP(methodRec, methodReq)
	if methodRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT /events status = %d, want %d", methodRec.Code, http.StatusMethodNotAllowed)
	}

	notFoundReq := httptest.NewRequest(http.MethodPost, "/events/"+event.ID+"/unknown", nil)
	notFoundRec := httptest.NewRecorder()
	mux.ServeHTTP(notFoundRec, notFoundReq)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("POST /events/{id}/unknown status = %d, want %d", notFoundRec.Code, http.StatusNotFound)
	}

}

func TestGroupRoutesHTTP(t *testing.T) {
	ctx := context.Background()
	pool := openIntegrationPool(t)

	tagsRepo := storage.NewEventTagsRepo()
	eventsRepo := storage.NewEventsRepo(pool, tagsRepo)
	groupRepo := storage.NewGroupRepo(pool)
	metrics := lib.NewMetrics()
	relayPriv, relayPub := generateKeypair(t)
	projection := services.NewGroupProjectionService(groupRepo, eventsRepo, relayPub, relayPriv, services.NewGroupVettingService(groupRepo), metrics)

	group := models.Group{
		GroupID:      "group-routes",
		Name:         "Group Routes",
		CreatedAt:    100,
		CreatedBy:    "owner-pub",
		UpdatedAt:    100,
		UpdatedBy:    "owner-pub",
		IsPrivate:    true,
		IsRestricted: true,
	}
	if err := groupRepo.UpsertGroup(ctx, group); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if err := groupRepo.UpsertRole(ctx, models.GroupRole{
		GroupID:     group.GroupID,
		RoleName:    "admin",
		Description: "admins",
		Permissions: []string{models.PermissionAdmin},
		CreatedAt:   101,
		CreatedBy:   "owner-pub",
		UpdatedAt:   101,
		UpdatedBy:   "owner-pub",
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := groupRepo.UpsertMember(ctx, models.GroupMember{
		GroupID:  group.GroupID,
		PubKey:   "member-a",
		AddedAt:  102,
		AddedBy:  "owner-pub",
		RoleName: "admin",
	}); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	if err := groupRepo.UpsertBan(ctx, models.GroupBan{
		GroupID:   group.GroupID,
		PubKey:    "banned-a",
		Reason:    "spam",
		BannedAt:  103,
		BannedBy:  "owner-pub",
		ExpiresAt: 0,
	}); err != nil {
		t.Fatalf("seed ban: %v", err)
	}
	if err := groupRepo.UpsertInvite(ctx, models.GroupInvite{
		GroupID:       group.GroupID,
		Code:          "code-1",
		MaxUsageCount: 3,
		UsageCount:    0,
		CreatedAt:     104,
		CreatedBy:     "owner-pub",
	}); err != nil {
		t.Fatalf("seed invite: %v", err)
	}

	mux := http.NewServeMux()
	relayhttp.RegisterGroupRoutes(mux, relayhttp.GroupRoutes{
		Repo:              groupRepo,
		ProjectionService: projection,
		Logger:            lib.NewLogger("ERROR"),
	})

	for _, path := range []string{
		"/groups",
		"/groups/" + group.GroupID,
		"/groups/" + group.GroupID + "/members",
		"/groups/" + group.GroupID + "/roles",
		"/groups/" + group.GroupID + "/bans",
		"/groups/" + group.GroupID + "/invites",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d body=%s", path, rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	joinBody, _ := json.Marshal(models.GroupJoinRequest{PubKey: "joiner-a"})
	joinReq := httptest.NewRequest(http.MethodPost, "/groups/"+group.GroupID+"/join-requests", bytes.NewReader(joinBody))
	joinRec := httptest.NewRecorder()
	mux.ServeHTTP(joinRec, joinReq)
	if joinRec.Code != http.StatusAccepted {
		t.Fatalf("POST join-requests status = %d, want %d body=%s", joinRec.Code, http.StatusAccepted, joinRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/groups/"+group.GroupID+"/join-requests/joiner-a/approve?approved_by=owner-pub", nil)
	approveRec := httptest.NewRecorder()
	mux.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusAccepted {
		t.Fatalf("POST approve status = %d, want %d body=%s", approveRec.Code, http.StatusAccepted, approveRec.Body.String())
	}
	roleName, exists, err := groupRepo.GetMemberRole(ctx, group.GroupID, "joiner-a")
	if err != nil || !exists || roleName != "member" {
		t.Fatalf("approved joiner role = (%q,%v,%v), want (member,true,nil)", roleName, exists, err)
	}

	missingApproverReq := httptest.NewRequest(http.MethodPost, "/groups/"+group.GroupID+"/join-requests/joiner-b/approve", nil)
	missingApproverRec := httptest.NewRecorder()
	mux.ServeHTTP(missingApproverRec, missingApproverReq)
	if missingApproverRec.Code != http.StatusBadRequest {
		t.Fatalf("approve without approver status = %d, want %d", missingApproverRec.Code, http.StatusBadRequest)
	}

	methodReq := httptest.NewRequest(http.MethodGet, "/groups/"+group.GroupID+"/join-requests", nil)
	methodRec := httptest.NewRecorder()
	mux.ServeHTTP(methodRec, methodReq)
	if methodRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET join-requests status = %d, want %d", methodRec.Code, http.StatusMethodNotAllowed)
	}

	notFoundReq := httptest.NewRequest(http.MethodGet, "/groups/"+group.GroupID+"/unknown", nil)
	notFoundRec := httptest.NewRecorder()
	mux.ServeHTTP(notFoundRec, notFoundReq)
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("GET unknown subroute status = %d, want %d", notFoundRec.Code, http.StatusNotFound)
	}
}
