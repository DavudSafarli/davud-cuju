package main

import (
	"context"
	"sort"
	"sync"
	"time"
)

type InMemStorage struct {
	scoreEventsMu sync.RWMutex
	// scoreEvents is the list of all deduplicatedscore events
	scoreEvents []ScoreEvent
	// eventIDs for fast duplication check
	eventIDs map[string]bool // Track EventIDs for duplicate detection
	// processedEvents is for tracking which events have been processed by MarkScoreEventsAsProcessed func
	processedEvents map[string]bool

	talentScoresMu sync.RWMutex
	// map of talentID to its scores
	talentScores map[TalentID][]TalentScore

	leaderboardMu sync.RWMutex
	// leaderboard is the sorted list of talent ranks by score, deduped by TalentID with max score.
	// too lazy to implement skip-list, therefore I go with eventual consistency approach.
	// this field will be recalculated once every N seconds from the talentRanks map.
	leaderboard []TalentRank
	// talentLeaderboardIndex is the map of talentID to its rank in the s.leaderboard.
	// Assuming that we'll have more reads than writes, this map provides a fast way of lookup.
	// This field is also recalculated once every N seconds same as leaderboard.
	talentLeaderboardIndex map[TalentID]int
}

// refreshInterval specifies how often to refresh the leaderboard(default is 1 seconds)
func NewInMemStorage(refreshInterval time.Duration) *InMemStorage {
	storage := &InMemStorage{
		eventIDs:        make(map[string]bool),
		processedEvents: make(map[string]bool),
		talentScores:    make(map[TalentID][]TalentScore),
		leaderboard:     make([]TalentRank, 0),
	}

	if refreshInterval == 0 {
		refreshInterval = 1 * time.Second
	}

	go storage.startPeriodicRefresh(refreshInterval)
	return storage
}

// SaveScoreEvent stores a score event
// Returns true if the event was saved, false if it was a duplicate
func (s *InMemStorage) SaveScoreEvent(ctx context.Context, event ScoreEvent) (bool, error) {
	s.scoreEventsMu.Lock()
	defer s.scoreEventsMu.Unlock()

	if s.eventIDs[event.EventID] {
		return false, nil
	}

	// Mark EventID as seen and save the event
	s.eventIDs[event.EventID] = true
	s.scoreEvents = append(s.scoreEvents, event)
	return true, nil
}

// ConsumeScoreEvents retrieves unprocessed score events up to the specified limit
func (s *InMemStorage) ConsumeScoreEvents(ctx context.Context, limit int) ([]ScoreEvent, error) {
	s.scoreEventsMu.RLock()
	defer s.scoreEventsMu.RUnlock()

	var unprocessed []ScoreEvent
	for _, event := range s.scoreEvents {
		if !s.processedEvents[event.EventID] {
			unprocessed = append(unprocessed, event)
			if len(unprocessed) >= limit {
				break
			}
		}
	}

	return unprocessed, nil
}

// MarkScoreEventsAsProcessed marks the given events as processed
func (s *InMemStorage) MarkScoreEventsAsProcessed(ctx context.Context, events []ScoreEvent) error {
	s.scoreEventsMu.Lock()
	defer s.scoreEventsMu.Unlock()

	for _, event := range events {
		s.processedEvents[event.EventID] = true
	}

	return nil
}

// SaveTalentScore stores a talent score by appending to the talent's score list
func (s *InMemStorage) SaveTalentScore(ctx context.Context, talentScore TalentScore) error {
	s.talentScoresMu.Lock()
	defer s.talentScoresMu.Unlock()

	s.talentScores[talentScore.TalentID] = append(s.talentScores[talentScore.TalentID], talentScore)

	return nil
}

func (s *InMemStorage) FindTalentRank(ctx context.Context, talentID TalentID) (TalentRank, bool, error) {
	s.leaderboardMu.RLock()
	defer s.leaderboardMu.RUnlock()

	leaderboardIndex, ok := s.talentLeaderboardIndex[talentID]
	if !ok {
		return TalentRank{}, false, nil
	}

	return s.leaderboard[leaderboardIndex], true, nil
}

func (s *InMemStorage) GetTopRankedTalents(ctx context.Context, limit int) ([]TalentRank, error) {
	s.leaderboardMu.RLock()
	defer s.leaderboardMu.RUnlock()

	if limit > len(s.leaderboard) {
		limit = len(s.leaderboard)
	}

	ranks := make([]TalentRank, limit)
	copy(ranks, s.leaderboard[:limit])

	return ranks, nil
}

// startPeriodicRefresh starts a goroutine that refreshes the leaderboard periodically
func (s *InMemStorage) startPeriodicRefresh(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.refreshLeaderboard()
	}
}

// refreshLeaderboard goes through all talent scores, and builds a new leaderboard.
// It also builds a new talentLeaderboardIndex map, which is used to quickly find a talent's rank in the leaderboard.
func (s *InMemStorage) refreshLeaderboard() {
	newLeaderboard := make([]TalentRank, 0, len(s.talentScores))

	s.talentScoresMu.RLock()
	for _, scores := range s.talentScores {
		var bestTalentScore TalentScore
		for _, score := range scores {
			if score.Score > bestTalentScore.Score {
				bestTalentScore = score
			}
		}

		if bestTalentScore.Score == 0 {
			continue
		}
		newLeaderboard = append(newLeaderboard, TalentRank{
			TalentID:    bestTalentScore.TalentID,
			TalentScore: bestTalentScore,
			Rank:        len(newLeaderboard) + 1,
		})
	}
	s.talentScoresMu.RUnlock()

	// Sort by score in descending order (highest score at index 0)
	sort.Slice(newLeaderboard, func(i, j int) bool {
		return newLeaderboard[i].TalentScore.Score > newLeaderboard[j].TalentScore.Score
	})

	newTalentLeaderboardIndex := make(map[TalentID]int)

	// set the rank field for each item in the leaderboard.
	// and populate the talentLeaderboardIndex map.
	for leaderboardIndex, talentRank := range newLeaderboard {
		talentRank.Rank = leaderboardIndex + 1
		newLeaderboard[leaderboardIndex] = talentRank

		newTalentLeaderboardIndex[talentRank.TalentID] = leaderboardIndex
	}

	s.leaderboardMu.Lock()
	s.leaderboard = newLeaderboard
	s.talentLeaderboardIndex = newTalentLeaderboardIndex
	s.leaderboardMu.Unlock()
}
