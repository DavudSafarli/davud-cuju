package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type CreateEventRequest struct {
	EventID   string    `json:"event_id"`
	TalentID  string    `json:"talent_id"`
	RawMetric int       `json:"raw_metric"`
	Skill     string    `json:"skill"`
	Timestamp time.Time `json:"ts"`
}

type LeaderboardResponse struct {
	Talents []TalentRankResponse `json:"talents"`
}

type TalentRankResponse struct {
	Rank     int    `json:"rank"`
	TalentID string `json:"talent_id"`
	Score    int    `json:"score"`
}

type GetTalentRankResponse struct {
	Rank     int    `json:"rank"`
	TalentID string `json:"talent_id"`
	Score    int    `json:"score"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

func (h *HTTPHandler) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /events", h.CreateEventHandler)
	mux.HandleFunc("GET /leaderboard", h.GetLeaderboardHandler)
	mux.HandleFunc("GET /rank/{talent_id}", h.GetTalentRankHandler)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	return mux
}

func (h *HTTPHandler) CreateEventHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	skill := Skill(req.Skill)
	if skill != SkillDribble && skill != SkillShoot && skill != SkillPass {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid skill", "skill must be one of: dribble, shoot, pass")
		return
	}

	event := ScoreEvent{
		EventID:     req.EventID,
		TalentID:    TalentID(req.TalentID),
		Skill:       skill,
		MetricValue: req.RawMetric,
		Timestamp:   req.Timestamp,
	}

	saved, err := h.service.SaveScoreEvent(r.Context(), event)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to save event", err.Error())
		return
	}

	if saved {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) GetLeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default limit
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid limit parameter", "limit must be a positive integer")
			return
		}
	}

	talents, err := h.service.GetTopTalents(r.Context(), limit)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get leaderboard", err.Error())
		return
	}

	response := LeaderboardResponse{
		Talents: make([]TalentRankResponse, len(talents)),
	}

	for i, talent := range talents {
		response.Talents[i] = TalentRankResponse{
			Rank:     talent.Rank,
			TalentID: string(talent.TalentID),
			Score:    talent.TalentScore.Score,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) GetTalentRankHandler(w http.ResponseWriter, r *http.Request) {
	talentID := r.PathValue("talent_id")
	if talentID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing talent_id", "talent_id is required in the URL path")
		return
	}

	talentRank, err := h.service.GetTalentRank(r.Context(), TalentID(talentID))
	if err != nil {
		if err == ErrTalentNotFound {
			writeErrorResponse(w, http.StatusNotFound, "Talent not found", fmt.Sprintf("Talent with ID '%s' not found in leaderboard", talentID))
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get talent rank", err.Error())
		return
	}

	response := GetTalentRankResponse{
		Rank:     talentRank.Rank,
		TalentID: string(talentRank.TalentID),
		Score:    talentRank.TalentScore.Score,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions
func writeErrorResponse(w http.ResponseWriter, statusCode int, error, message string) {
	response := ErrorResponse{
		Error:   error,
		Message: message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
