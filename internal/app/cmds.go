package app

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/yourname/eitango/internal/quiz"
	"github.com/yourname/eitango/internal/session"
	"github.com/yourname/eitango/internal/srs"
	"github.com/yourname/eitango/internal/store"
)

func loadHomeCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		home, err := st.LoadHomeSnapshot(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		snapshot, err := st.LoadStatsSnapshot(ctx)
		if err != nil {
			return errMsg{err: err}
		}
		return homeLoadedMsg{Home: home, Stats: snapshot}
	}
}

func loadStatsCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		snapshot, err := st.LoadStatsSnapshot(context.Background())
		if err != nil {
			return errMsg{err: err}
		}
		return statsLoadedMsg{Snapshot: snapshot}
	}
}

func sessionCmd(st *store.Store, svc *quiz.Service, mode string, replaceActive bool, recent []int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if replaceActive {
			if err := st.AbandonActiveSession(ctx); err != nil {
				return errMsg{err: err}
			}
		}

		if !replaceActive {
			record, items, err := st.LoadActiveRuntime(ctx)
			if err != nil {
				return errMsg{err: err}
			}
			if record != nil {
				runtime := session.NewRuntime(*record, items)
				question, err := buildCurrentQuestion(ctx, svc, runtime, recent)
				if err != nil {
					return errMsg{err: err}
				}
				return sessionLoadedMsg{Runtime: runtime, Question: question}
			}
		}

		dueWords, err := st.ListDueWords(ctx, session.DefaultQuestionCount)
		if err != nil {
			return errMsg{err: err}
		}

		var itemsPlan []store.SessionItemPlan
		switch mode {
		case store.ModeReview:
			plan := session.MakePlan(session.DefaultQuestionCount, len(dueWords), 0, store.ModeReview)
			itemsPlan = session.BuildSessionItems(dueWords[:plan.ReviewCount], nil)
		default:
			dueIDs := make([]int64, 0, len(dueWords))
			for _, word := range dueWords {
				dueIDs = append(dueIDs, word.ID)
			}
			newWords, err := st.ListNewWords(ctx, session.DefaultQuestionCount, dueIDs)
			if err != nil {
				return errMsg{err: err}
			}

			plan := session.MakePlan(session.DefaultQuestionCount, len(dueWords), len(newWords), store.ModeLearn)
			reviewWords := dueWords[:plan.ReviewCount]
			newSelection := newWords[:plan.NewCount]
			itemsPlan = session.BuildSessionItems(reviewWords, newSelection)
		}

		if len(itemsPlan) == 0 {
			return errMsg{err: fmt.Errorf("no words available for this session")}
		}

		record, items, err := st.CreateSession(ctx, mode, itemsPlan)
		if err != nil {
			return errMsg{err: err}
		}
		runtime := session.NewRuntime(record, items)
		question, err := buildCurrentQuestion(ctx, svc, runtime, recent)
		if err != nil {
			return errMsg{err: err}
		}
		return sessionLoadedMsg{Runtime: runtime, Question: question}
	}
}

func submitAnswerCmd(st *store.Store, svc *quiz.Service, runtime *session.Runtime, feedback quiz.Feedback, rating srs.Rating, recent []int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		item, ok := runtime.CurrentItem()
		if !ok {
			return errMsg{err: fmt.Errorf("no active question")}
		}

		record, items, err := st.SaveAnswer(ctx, store.ReviewEvent{
			SessionID:      runtime.Session.ID,
			ItemOrdinal:    item.Ordinal,
			WordID:         item.WordID,
			Kind:           item.Kind,
			SelectedChoice: feedback.SelectedIndex,
			CorrectChoice:  feedback.Question.CorrectIndex,
			IsCorrect:      feedback.Correct,
			Rating:         rating,
			AnsweredAt:     time.Now().UTC(),
			ResponseMS:     feedback.ResponseMS,
		})
		if err != nil {
			return errMsg{err: err}
		}

		nextRuntime := session.NewRuntime(record, items)
		if record.Status == store.SessionStatusCompleted {
			summary, err := st.LoadSessionSummary(ctx, record.ID)
			if err != nil {
				return errMsg{err: err}
			}
			return answerSavedMsg{Runtime: nextRuntime, Summary: &summary, Status: "Saved"}
		}

		question, err := buildCurrentQuestion(ctx, svc, nextRuntime, recent)
		if err != nil {
			return errMsg{err: err}
		}
		return answerSavedMsg{Runtime: nextRuntime, NextQuestion: &question, Status: "Saved"}
	}
}

func buildCurrentQuestion(ctx context.Context, svc *quiz.Service, runtime *session.Runtime, recent []int64) (quiz.Question, error) {
	item, ok := runtime.CurrentItem()
	if !ok {
		return quiz.Question{}, fmt.Errorf("no pending question")
	}
	return svc.BuildQuestion(ctx, item, runtime.Total(), recent)
}
