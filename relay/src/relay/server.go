package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/fiatjaf/khatru"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nbd-wtf/go-nostr"

	"s-city/src/lib"
	"s-city/src/models"
	"s-city/src/services"
	"s-city/src/storage"
)

// Server wires the relay runtime and its HTTP handlers.
type Server struct {
	cfg        lib.Config
	logger     *slog.Logger
	metrics    *lib.Metrics
	db         *pgxpool.Pool
	httpServer *http.Server
}

func NewServer(ctx context.Context, cfg lib.Config) (*Server, error) {
	logger := lib.NewLogger(cfg.LogLevel)
	metrics := lib.NewMetrics()

	db, err := storage.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := applyMigrations(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	tagsRepo := storage.NewEventTagsRepo()
	eventsRepo := storage.NewEventsRepo(db, tagsRepo)
	groupRepo := storage.NewGroupRepo(db)

	validator := services.NewValidator(cfg.MaxEventSkew)
	abuseControls := services.NewAbuseControls(cfg.RateLimitBurst, cfg.RateLimitPerMinute, cfg.DefaultPowBits)
	vettingService := services.NewGroupVettingService(groupRepo)
	projectionService := services.NewGroupProjectionService(groupRepo, eventsRepo, cfg.RelayPubKey, cfg.RelayPrivKey, vettingService, metrics)
	ingestService := services.NewEventIngestService(eventsRepo, validator, abuseControls, projectionService, metrics, cfg.RelayPubKey)
	queryService := services.NewEventQueryService(eventsRepo)
	deleteService := services.NewEventDeleteService(eventsRepo, projectionService, metrics)
	khatruRelay := khatru.NewRelay()

	wireKhatruHooks(khatruRelay, ingestService, queryService, deleteService)

	mux := khatruRelay.Router()
	RegisterEventRoutes(mux, EventRoutes{
		IngestService: ingestService,
		QueryService:  queryService,
		DeleteService: deleteService,
		Logger:        logger,
	})
	RegisterGroupRoutes(mux, GroupRoutes{
		Repo:              groupRepo,
		ProjectionService: projectionService,
		Logger:            logger,
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metrics.Snapshot())
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           khatruRelay,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		cfg:        cfg,
		logger:     logger,
		metrics:    metrics,
		db:         db,
		httpServer: httpServer,
	}, nil
}

func (s *Server) Start() error {
	s.logger.Info("relay server starting", "addr", s.cfg.HTTPAddr)
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	defer s.db.Close()
	return s.httpServer.Shutdown(ctx)
}

func applyMigrations(ctx context.Context, db *pgxpool.Pool) error {
	files := make([]string, 0)
	if err := filepath.WalkDir("src/storage/migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".sql" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk migration files: %w", err)
	}
	sort.Strings(files)

	for _, path := range files {
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", path, err)
		}
		if _, err := db.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %s: %w", path, err)
		}
	}
	return nil
}

func wireKhatruHooks(
	relay *khatru.Relay,
	ingestService *services.EventIngestService,
	queryService *services.EventQueryService,
	deleteService *services.EventDeleteService,
) {
	relay.StoreEvent = append(relay.StoreEvent, func(ctx context.Context, event *nostr.Event) error {
		modelEvent := modelEventFromNostr(event)
		err := ingestService.Ingest(ctx, modelEvent)
		if errors.Is(err, services.ErrDuplicateEvent) {
			return eventstore.ErrDupEvent
		}
		return err
	})

	relay.DeleteEvent = append(relay.DeleteEvent, func(ctx context.Context, target *nostr.Event) error {
		return deleteService.DeleteEvent(ctx, models.DeletedEvent{
			EventID:   target.ID,
			DeletedAt: time.Now().Unix(),
			DeletedBy: target.PubKey,
			Reason:    "relay delete",
		})
	})

	relay.QueryEvents = append(relay.QueryEvents, func(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
		events, err := queryService.QueryNostrFilter(ctx, filter)
		if err != nil {
			return nil, err
		}

		ch := make(chan *nostr.Event, len(events))
		for _, event := range events {
			ch <- nostrEventFromModel(event)
		}
		close(ch)
		return ch, nil
	})
}

func modelEventFromNostr(event *nostr.Event) models.Event {
	tags := make([][]string, 0, len(event.Tags))
	for _, tag := range event.Tags {
		tags = append(tags, append([]string(nil), tag...))
	}
	return models.Event{
		ID:        event.ID,
		PubKey:    event.PubKey,
		CreatedAt: int64(event.CreatedAt),
		Kind:      event.Kind,
		Tags:      tags,
		Content:   event.Content,
		Sig:       event.Sig,
	}
}

func nostrEventFromModel(event models.Event) *nostr.Event {
	tags := make(nostr.Tags, 0, len(event.Tags))
	for _, tag := range event.Tags {
		tags = append(tags, append(nostr.Tag(nil), tag...))
	}
	return &nostr.Event{
		ID:        event.ID,
		PubKey:    event.PubKey,
		CreatedAt: nostr.Timestamp(event.CreatedAt),
		Kind:      event.Kind,
		Tags:      tags,
		Content:   event.Content,
		Sig:       event.Sig,
	}
}
