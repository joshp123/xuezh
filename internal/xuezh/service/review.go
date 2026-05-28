package service

import (
	"database/sql"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/srs"
)

type ReviewItem struct {
	ItemID     string
	DueAt      string
	ReviewType string
}

type StartReviewResult struct {
	RecallItems        []ReviewItem
	PronunciationItems []ReviewItem
	GeneratedAt        string
}

type GradeReviewOptions struct {
	ItemID             string
	RecallGrade        *int
	PronunciationGrade *int
	NextDue            string
	Rule               string
}

type GradeReviewResult struct {
	ItemID                   string
	RecallGrade              *int
	RecallNextDue            *string
	RecallRuleApplied        *string
	PronunciationGrade       *int
	PronunciationNextDue     *string
	PronunciationRuleApplied *string
}

type BuryReviewResult struct {
	ItemID  string
	Reason  string
	NextDue string
}

type SRSPreviewResult struct {
	Days     int
	Forecast map[string]map[string]int
}

func (App) StartReview(limit int, now time.Time) (StartReviewResult, error) {
	recallItems, err := srs.ListDueItems(limit, now, "recall")
	if err != nil {
		return StartReviewResult{}, err
	}
	pronunciationItems, err := srs.ListDueItems(limit, now, "pronunciation")
	if err != nil {
		return StartReviewResult{}, err
	}
	return StartReviewResult{
		RecallItems:        reviewItems(recallItems),
		PronunciationItems: reviewItems(pronunciationItems),
		GeneratedAt:        clock.FormatISO(now),
	}, nil
}

func reviewItems(items []srs.DueItem) []ReviewItem {
	result := make([]ReviewItem, 0, len(items))
	for _, item := range items {
		result = append(result, ReviewItem{
			ItemID:     item.ItemID,
			DueAt:      item.DueAt,
			ReviewType: item.ReviewType,
		})
	}
	return result
}

func (App) GradeReview(opts GradeReviewOptions, now time.Time) (GradeReviewResult, error) {
	recallGrade := opts.RecallGrade

	var recallDueAt *string
	var recallRule *string
	if recallGrade != nil {
		dueAt, appliedRule, err := srs.ScheduleNextDue(*recallGrade, now, opts.Rule, opts.NextDue)
		if err != nil {
			return GradeReviewResult{}, err
		}
		recallDueAt = &dueAt
		if appliedRule != "" {
			recallRule = &appliedRule
		}
	}

	var pronDueAt *string
	var pronRule *string
	if opts.PronunciationGrade != nil {
		pronNextDue := ""
		if recallGrade == nil {
			pronNextDue = opts.NextDue
		}
		dueAt, appliedRule, err := srs.ScheduleNextDue(*opts.PronunciationGrade, now, opts.Rule, pronNextDue)
		if err != nil {
			return GradeReviewResult{}, err
		}
		pronDueAt = &dueAt
		if appliedRule != "" {
			pronRule = &appliedRule
		}
	}

	if err := withReviewTx(func(tx *sql.Tx) error {
		if err := srs.UpsertKnowledgeTx(tx, opts.ItemID, recallDueAt, recallGrade, pronDueAt, opts.PronunciationGrade, now); err != nil {
			return err
		}
		if recallGrade != nil {
			payload := map[string]any{
				"review_type": "recall",
				"grade":       *recallGrade,
				"rule":        recallRule,
				"next_due":    recallDueAt,
			}
			if err := srs.RecordReviewEventTx(tx, opts.ItemID, "review.grade", payload, now); err != nil {
				return err
			}
		}
		if opts.PronunciationGrade != nil {
			payload := map[string]any{
				"review_type": "pronunciation",
				"grade":       *opts.PronunciationGrade,
				"rule":        pronRule,
				"next_due":    pronDueAt,
			}
			if err := srs.RecordReviewEventTx(tx, opts.ItemID, "review.grade", payload, now); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return GradeReviewResult{}, err
	}

	result := GradeReviewResult{
		ItemID:                   opts.ItemID,
		RecallGrade:              recallGrade,
		RecallNextDue:            recallDueAt,
		RecallRuleApplied:        recallRule,
		PronunciationGrade:       opts.PronunciationGrade,
		PronunciationNextDue:     pronDueAt,
		PronunciationRuleApplied: pronRule,
	}
	return result, nil
}

func (App) BuryReview(itemID, reason string, now time.Time) (BuryReviewResult, error) {
	dueAt, _, err := srs.ScheduleNextDue(0, now, "leitner", "")
	if err != nil {
		return BuryReviewResult{}, err
	}
	if err := withReviewTx(func(tx *sql.Tx) error {
		if err := srs.UpsertKnowledgeTx(tx, itemID, &dueAt, nil, nil, nil, now); err != nil {
			return err
		}
		payload := map[string]any{"reason": reason, "next_due": dueAt}
		return srs.RecordReviewEventTx(tx, itemID, "review.bury", payload, now)
	}); err != nil {
		return BuryReviewResult{}, err
	}
	return BuryReviewResult{ItemID: itemID, Reason: reason, NextDue: dueAt}, nil
}

func (App) PreviewSRS(days int, now time.Time) (SRSPreviewResult, error) {
	recall, err := srs.PreviewDue(days, now, "recall")
	if err != nil {
		return SRSPreviewResult{}, err
	}
	pronunciation, err := srs.PreviewDue(days, now, "pronunciation")
	if err != nil {
		return SRSPreviewResult{}, err
	}
	return SRSPreviewResult{
		Days: days,
		Forecast: map[string]map[string]int{
			"recall":        recall,
			"pronunciation": pronunciation,
		},
	}, nil
}

func withReviewTx(fn func(*sql.Tx) error) error {
	dbPath, err := db.InitDB()
	if err != nil {
		return err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
