package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ProcessScoreEvent_GetTopTalents(t *testing.T) {
	t.Run("process single score event and get top talents", func(t *testing.T) {
		storage := NewInMemStorage(10 * time.Millisecond) // Fast refresh for testing
		scorer := NewLinearScorer()
		service := NewService(storage, scorer)

		// Create a score event
		scoreEvent := ScoreEvent{
			EventID:     "event-1",
			TalentID:    "talent-1",
			Skill:       SkillDribble,
			MetricValue: 100,
			Timestamp:   time.Now(),
		}

		// Save the score event
		exists, err := service.SaveScoreEvent(context.Background(), scoreEvent)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Start the background job in a goroutine
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go func() {
			service.ProcessScoreEvents(ctx, 10)
		}()

		// Use EventuallyWithT to wait for the background job to process the event
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			talents, err := service.GetTopTalents(context.Background(), 10)
			require.NoError(c, err)
			require.Len(c, talents, 1)
			assert.Equal(c, TalentID("talent-1"), talents[0].TalentID)
			assert.Equal(c, 100, talents[0].TalentScore.Score)
			assert.Equal(c, 1, talents[0].Rank)
		}, 2*time.Second, 50*time.Millisecond, "Expected talent to appear in leaderboard with correct score and rank")
	})

	t.Run("process multiple talents with multiple score events each", func(t *testing.T) {
		storage := NewInMemStorage(10 * time.Millisecond) // Fast refresh for testing
		scorer := NewLinearScorer()
		service := NewService(storage, scorer)

		// Create score events for 3 different talents, each with multiple events
		events := []ScoreEvent{
			// Talent 1: dribble=50, shoot=80 (max=80)
			{EventID: "event-1", TalentID: TalentID("talent-1"), Skill: SkillDribble, MetricValue: 50, Timestamp: time.Now()},
			{EventID: "event-2", TalentID: TalentID("talent-1"), Skill: SkillShoot, MetricValue: 80, Timestamp: time.Now()},

			// Talent 2: pass=60, dribble=40 (max=60)
			{EventID: "event-3", TalentID: TalentID("talent-2"), Skill: SkillPass, MetricValue: 60, Timestamp: time.Now()},
			{EventID: "event-4", TalentID: TalentID("talent-2"), Skill: SkillDribble, MetricValue: 40, Timestamp: time.Now()},

			// Talent 3: shoot=30, pass=70 (max=70)
			{EventID: "event-5", TalentID: TalentID("talent-3"), Skill: SkillShoot, MetricValue: 30, Timestamp: time.Now()},
			{EventID: "event-6", TalentID: TalentID("talent-3"), Skill: SkillPass, MetricValue: 70, Timestamp: time.Now()},
		}

		// Save all events
		for _, event := range events {
			exists, err := service.SaveScoreEvent(context.Background(), event)
			require.NoError(t, err)
			require.True(t, exists)
		}

		// Start the background job in a goroutine
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go func() {
			service.ProcessScoreEvents(ctx, 10)
		}()

		// Use EventuallyWithT to wait for all talents to appear in leaderboard with correct ranking
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			talents, err := service.GetTopTalents(context.Background(), 10)
			require.NoError(c, err)
			require.Len(c, talents, 3)

			// Check ranking order (highest score first)
			// Talent 1 should be 1st (score=80)
			assert.Equal(c, TalentID("talent-1"), talents[0].TalentID)
			assert.Equal(c, 80, talents[0].TalentScore.Score)
			assert.Equal(c, 1, talents[0].Rank)

			// Talent 3 should be 2nd (score=70)
			assert.Equal(c, TalentID("talent-3"), talents[1].TalentID)
			assert.Equal(c, 70, talents[1].TalentScore.Score)
			assert.Equal(c, 2, talents[1].Rank)

			// Talent 2 should be 3rd (score=60)
			assert.Equal(c, TalentID("talent-2"), talents[2].TalentID)
			assert.Equal(c, 60, talents[2].TalentScore.Score)
			assert.Equal(c, 3, talents[2].Rank)
		}, 3*time.Second, 50*time.Millisecond, "Expected all 3 talents to appear in leaderboard with correct ranking based on max scores")
	})
}

func TestService_GetTalentRank(t *testing.T) {
	t.Run("get talent rank", func(t *testing.T) {
		storage := NewInMemStorage(10 * time.Millisecond)
		scorer := NewLinearScorer()
		service := NewService(storage, scorer)

		// Create some talents
		events := []ScoreEvent{
			{EventID: "event-1", TalentID: TalentID("talent-1"), Skill: SkillDribble, MetricValue: 50, Timestamp: time.Now()},
			{EventID: "event-2", TalentID: TalentID("talent-1"), Skill: SkillShoot, MetricValue: 80, Timestamp: time.Now()},
		}

		// Save all events
		for _, event := range events {
			exists, err := service.SaveScoreEvent(context.Background(), event)
			require.NoError(t, err)
			require.True(t, exists)
		}

		// Start the background job in a goroutine
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		go func() {
			service.ProcessScoreEvents(ctx, 10)
		}()

		// Use EventuallyWithT to wait for the background job to process the event
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			talent, err := service.GetTalentRank(context.Background(), TalentID("talent-1"))
			require.NoError(c, err)
			assert.Equal(c, TalentID("talent-1"), talent.TalentID)
			assert.Equal(c, 80, talent.TalentScore.Score)
			assert.Equal(c, 1, talent.Rank)
		}, 2*time.Second, 50*time.Millisecond, "Expected talent to appear in leaderboard with correct score and rank")
	})
}
