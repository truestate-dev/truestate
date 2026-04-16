// Package api implements the TrueState REST API handlers.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.local.vjinx.de/truestate/truestate/internal/db"
	"gitea.local.vjinx.de/truestate/truestate/internal/engine"
	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

// Router builds and returns the chi router with all routes mounted.
func Router(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		// Inventory
		r.Post("/inventories", handleCreateInventory(pool))
		r.Get("/inventories", handleListInventories(pool))
		r.Get("/inventories/{id}", handleGetInventory(pool))
		r.Post("/inventories/{id}/relation", handleSetRelation(pool))
		r.Get("/inventories/{id}/evaluations", handleListEvaluations(pool))

		// Evaluation — POST triggers + persists; GET retrieves a stored result
		r.Post("/evaluate/{id}", handleEvaluate(pool))
		r.Get("/evaluations/{id}", handleGetEvaluation(pool))

		// Sources
		r.Get("/sources", handleListSources(pool))
	})

	return r
}

func handleCreateInventory(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var inv model.Inventory
		if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := db.CreateInventory(r.Context(), pool, &inv); err != nil {
			slog.Error("create inventory", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to create inventory")
			return
		}
		writeJSON(w, http.StatusCreated, inv)
	}
}

func handleListInventories(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invs, err := db.ListInventories(r.Context(), pool)
		if err != nil {
			slog.Error("list inventories", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to list inventories")
			return
		}
		writeJSON(w, http.StatusOK, invs)
	}
}

func handleGetInventory(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		inv, err := db.GetInventory(r.Context(), pool, id)
		if err != nil {
			slog.Error("get inventory", "id", id, "err", err)
			writeError(w, http.StatusNotFound, "inventory not found")
			return
		}
		writeJSON(w, http.StatusOK, inv)
	}
}

func handleSetRelation(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hostID := chi.URLParam(r, "id")
		var body struct {
			GoldenInventoryID string `json:"golden_inventory_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := db.SetInventoryRelation(r.Context(), pool, hostID, body.GoldenInventoryID); err != nil {
			slog.Error("set relation", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to set relation")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleEvaluate(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		eval, err := engine.Evaluate(r.Context(), pool, id)
		if err != nil {
			slog.Error("evaluate", "id", id, "err", err)
			writeError(w, http.StatusInternalServerError, "evaluation failed")
			return
		}
		if err := db.SaveEvaluation(r.Context(), pool, eval); err != nil {
			slog.Error("save evaluation", "id", id, "err", err)
			writeError(w, http.StatusInternalServerError, "failed to persist evaluation")
			return
		}
		writeJSON(w, http.StatusCreated, eval)
	}
}

func handleGetEvaluation(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		eval, err := db.GetEvaluation(r.Context(), pool, id)
		if err != nil {
			slog.Error("get evaluation", "id", id, "err", err)
			writeError(w, http.StatusNotFound, "evaluation not found")
			return
		}
		writeJSON(w, http.StatusOK, eval)
	}
}

func handleListEvaluations(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		summaries, err := db.ListEvaluationsForInventory(r.Context(), pool, id)
		if err != nil {
			slog.Error("list evaluations", "inventory_id", id, "err", err)
			writeError(w, http.StatusInternalServerError, "failed to list evaluations")
			return
		}
		writeJSON(w, http.StatusOK, summaries)
	}
}

func handleListSources(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sources, err := db.ListSourceStatus(r.Context(), pool)
		if err != nil {
			slog.Error("list sources", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to list sources")
			return
		}
		writeJSON(w, http.StatusOK, sources)
	}
}

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json response", "err", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
