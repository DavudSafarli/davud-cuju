package main

import (
	"context"
	"errors"
	"log"
	"time"
)

var ErrTalentNotFound = errors.New("talent not found")
var ErrDuplicateScoreEvent = errors.New("duplicate score event")

type TalentID string

type Skill string

const (
	SkillDribble Skill = "dribble"
	SkillShoot   Skill = "shoot"
	SkillPass    Skill = "pass"
)

type ScoreEvent struct {
	EventID     string
	TalentID    TalentID
	Skill       Skill
	MetricValue int
	Timestamp   time.Time
}

// TalentScore is a score calculated for a talent for a specific skill.
type TalentScore struct {
	TalentID TalentID
	Skill    Skill
	Score    int

	// EventID is reference to the ScoreEvent ID that was used to calculate the score
	EventID string
}

// TalentRank shows Talent's rank in the leaderboard, specific TalentScore that determined this ranking.
type TalentRank struct {
	TalentID    TalentID
	TalentScore TalentScore

	Rank int
}

type Storage interface {
	// SaveScoreEvent saves a score event; returns true if the event was saved, false if it was a duplicate
	SaveScoreEvent(ctx context.Context, event ScoreEvent) (bool, error)
	// ConsumeScoreEvents returns score events in the order of insertion.
	// Once an event is marked as Processed by #MarkScoreEventsAsProcessed,
	// it won't be returned in the next call ConsumeScoreEvents call.
	ConsumeScoreEvents(ctx context.Context, limit int) ([]ScoreEvent, error)
	MarkScoreEventsAsProcessed(ctx context.Context, events []ScoreEvent) error

	SaveTalentScore(ctx context.Context, talentScore TalentScore) error

	GetTopRankedTalents(ctx context.Context, limit int) ([]TalentRank, error)
	FindTalentRank(ctx context.Context, talentID TalentID) (TalentRank, bool, error)
}

type Scorer interface {
	CalculateScore(ctx context.Context, skill Skill, metricValue int) (int, error)
}

type Service struct {
	storage Storage
	scorer  Scorer
}

func NewService(storage Storage, scorer Scorer) *Service {
	return &Service{
		storage: storage,
		scorer:  scorer,
	}
}

// SaveScoreEvent saves a score event; returns true if the event was saved, false if it was a duplicate
func (s *Service) SaveScoreEvent(ctx context.Context, event ScoreEvent) (bool, error) {
	IncScoreEventsTotal()

	saved, err := s.storage.SaveScoreEvent(ctx, event)
	if err != nil {
		return false, err
	}

	if !saved {
		IncScoreEventsDuplicate()
	}

	return saved, nil
}

func (s *Service) GetTopTalents(ctx context.Context, limit int) ([]TalentRank, error) {
	talentRanks, err := s.storage.GetTopRankedTalents(ctx, limit)
	if err != nil {
		return nil, err
	}

	return talentRanks, nil
}

func (s *Service) GetTalentRank(ctx context.Context, talentID TalentID) (TalentRank, error) {
	talentRank, found, err := s.storage.FindTalentRank(ctx, talentID)
	if err != nil {
		return TalentRank{}, err
	}
	if !found {
		return TalentRank{}, ErrTalentNotFound
	}
	return talentRank, nil
}

// ProcessScoreEvents consumes the score events, calculates the score for each and saves them.
// There's a room for concurrency here, but I'll leave it for now.
func (s *Service) ProcessScoreEvents(ctx context.Context, limit int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		events, err := s.storage.ConsumeScoreEvents(ctx, limit)
		if err != nil {
			log.Printf("Error consuming score events: %v", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
			continue
		}

		var processedEvents []ScoreEvent
		for _, event := range events {
			score, err := s.scorer.CalculateScore(ctx, event.Skill, event.MetricValue)
			if err != nil {
				log.Printf("Error calculating score for event %s: %v", event.EventID, err)
				continue
			}

			talentScore := TalentScore{
				TalentID: event.TalentID,
				Skill:    event.Skill,
				Score:    score,
				EventID:  event.EventID,
			}

			err = s.storage.SaveTalentScore(ctx, talentScore)
			if err != nil {
				log.Printf("Error saving talent score for event %s: %v", event.EventID, err)
				continue
			}

			processedEvents = append(processedEvents, event)
		}

		if len(processedEvents) > 0 {
			err = s.storage.MarkScoreEventsAsProcessed(ctx, processedEvents)
			if err != nil {
				log.Printf("Error marking events as processed: %v", err)
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
