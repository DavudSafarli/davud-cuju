package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type WeightBasedScorer struct {
	skillWeights map[Skill]int
}

func NewWeightBasedScorer(skillWeights map[Skill]int) *WeightBasedScorer {
	return &WeightBasedScorer{
		skillWeights: skillWeights,
	}
}

func (s *WeightBasedScorer) CalculateScore(ctx context.Context, skill Skill, metricValue int) (int, error) {
	// Add random delay between 80-150ms
	delay := time.Duration(80+rand.Intn(71)) * time.Millisecond
	time.Sleep(delay)

	weight, ok := s.skillWeights[skill]
	if !ok {
		return 0, fmt.Errorf("skill %s not found", skill)
	}
	return metricValue * weight, nil
}

type LinearScorer struct{}

func NewLinearScorer() *LinearScorer {
	return &LinearScorer{}
}

// CalculateScore calculates a score based on skill and metric value
func (s *LinearScorer) CalculateScore(ctx context.Context, skill Skill, metricValue int) (int, error) {
	return metricValue, nil
}
