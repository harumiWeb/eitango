package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango/internal/audio"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/quiz"
	"github.com/harumiWeb/eitango/internal/session"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/updatecheck"
)

type sessionRequest struct {
	Mode                string
	AnswerMode          string
	WriteModeDifficulty string
	AllowReviewFallback bool
	ReplaceActive       bool
	Plan                session.PlanOptions
}

func loadHomeCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := st.AbandonInfiniteReviewSessions(ctx); err != nil {
			return errMsg{err: err}
		}
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

func updateCheckCmd(service updatecheck.Service, currentVersion string) tea.Cmd {
	if service == nil {
		return nil
	}
	return func() tea.Msg {
		result, _ := service.CheckNow(context.Background(), currentVersion)
		return updateCheckedMsg{Result: result}
	}
}

func sessionCmd(st *store.Store, svc *quiz.Service, request sessionRequest, recent []int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		mode := request.Mode
		if mode == "" {
			mode = store.ModeLearn
		}
		answerMode := store.NormalizeAnswerMode(request.AnswerMode)
		writeModeDifficulty := config.NormalizeWriteModeDifficulty(request.WriteModeDifficulty)
		options := request.Plan.Normalize()

		if !request.ReplaceActive {
			if err := st.AbandonInfiniteReviewSessions(ctx); err != nil {
				return errMsg{err: err}
			}
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

		dueWords, err := st.ListDueWords(ctx, options.QuestionCount)
		if err != nil {
			return errMsg{err: err}
		}

		var itemsPlan []store.SessionItemPlan
		switch mode {
		case store.ModeReview:
			reviewWords := dueWords
			sessionMode := store.ModeReview
			if len(reviewWords) == 0 {
				reviewWords, err = st.ListReviewedWordsRandom(ctx, options.QuestionCount)
				if err != nil {
					return errMsg{err: err}
				}
				if len(reviewWords) > 0 && !request.AllowReviewFallback {
					request.AllowReviewFallback = true
					return reviewFallbackPromptMsg{Request: request}
				}
				if len(reviewWords) > 0 {
					sessionMode = store.ModeReviewInfinite
				}
			}
			plan := session.MakePlan(options, len(reviewWords), 0, store.ModeReview)
			itemsPlan = session.BuildSessionItems(reviewWords[:plan.ReviewCount], nil)
			mode = sessionMode
		default:
			dueIDs := make([]int64, 0, len(dueWords))
			for _, word := range dueWords {
				dueIDs = append(dueIDs, word.ID)
			}
			var newWords []store.Word
			basicWritePoolEmpty := false
			if answerMode == store.AnswerModeWrite && writeModeDifficulty == config.WriteModeDifficultyBasic {
				newWords, err = st.ListWriteBasicCandidates(ctx, options.QuestionCount, dueIDs)
				basicWritePoolEmpty = len(newWords) == 0
			} else {
				newWords, err = st.ListNewWords(ctx, options.QuestionCount, dueIDs)
			}
			if err != nil {
				return errMsg{err: err}
			}

			plan := session.MakePlan(options, len(dueWords), len(newWords), mode)
			reviewWords := dueWords[:plan.ReviewCount]
			newSelection := newWords[:plan.NewCount]
			itemsPlan = session.BuildSessionItems(reviewWords, newSelection)

			if len(itemsPlan) == 0 && basicWritePoolEmpty {
				return errMsg{err: errors.New(i18n.T(i18n.StatusWriteBasicEmpty))}
			}
		}

		if len(itemsPlan) == 0 {
			return errMsg{err: fmt.Errorf("no words available for this session")}
		}

		replacedActive := false
		if request.ReplaceActive {
			if err := st.AbandonActiveSession(ctx); err != nil {
				return errMsg{err: err}
			}
			replacedActive = true
		}

		record, items, err := st.CreateSession(ctx, mode, answerMode, itemsPlan)
		if err != nil {
			return sessionStartErrMsg(st, err, replacedActive)
		}
		runtime := session.NewRuntime(record, items)
		question, err := buildCurrentQuestion(ctx, svc, runtime, recent)
		if err != nil {
			return sessionStartErrMsg(st, err, replacedActive)
		}
		return sessionLoadedMsg{Runtime: runtime, Question: question}
	}
}

func abandonInfiniteReviewAndQuitCmd(st *store.Store) tea.Cmd {
	return func() tea.Msg {
		if st == nil {
			return quitAfterCleanupMsg{}
		}
		if err := st.AbandonInfiniteReviewSessions(context.Background()); err != nil {
			return errMsg{err: err}
		}
		return quitAfterCleanupMsg{}
	}
}

func sessionStartErrMsg(st *store.Store, err error, reloadHome bool) tea.Msg {
	if !reloadHome {
		return errMsg{err: err}
	}

	ctx := context.Background()
	home, loadErr := st.LoadHomeSnapshot(ctx)
	if loadErr != nil {
		return errMsg{err: err}
	}
	msg := homeReloadedErrMsg{Home: home, err: err}
	snapshot, loadErr := st.LoadStatsSnapshot(ctx)
	if loadErr == nil {
		msg.Stats = &snapshot
	}
	return msg
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
			AnswerMode:     feedback.Question.AnswerMode,
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
			return answerSavedMsg{Runtime: nextRuntime, Summary: &summary, Status: i18n.T(i18n.StatusSaved)}
		}

		question, err := buildCurrentQuestion(ctx, svc, nextRuntime, recent)
		if err != nil {
			return errMsg{err: err}
		}
		return answerSavedMsg{Runtime: nextRuntime, NextQuestion: &question, Status: i18n.T(i18n.StatusSaved)}
	}
}

func saveSettingsCmd(path string, settings config.Settings, focusModeDisabled bool) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			return errMsg{err: fmt.Errorf("config path is not configured")}
		}
		if err := config.Save(path, settings); err != nil {
			return errMsg{err: err}
		}
		return settingsSavedMsg{
			Settings:          settings,
			FocusModeDisabled: focusModeDisabled,
		}
	}
}

func saveKeymapCmd(path string, settings config.Settings, focusModeDisabled bool) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			return errMsg{err: fmt.Errorf("config path is not configured")}
		}
		if err := config.Save(path, settings); err != nil {
			return errMsg{err: err}
		}
		return keymapSavedMsg{
			Settings:          settings,
			FocusModeDisabled: focusModeDisabled,
		}
	}
}

func speakCmd(speaker audio.Speaker, text string, fromAutoplay bool) tea.Cmd {
	return func() tea.Msg {
		if speaker == nil || !speaker.Enabled() {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := speaker.Speak(ctx, text); err != nil {
			return audioErrMsg{fromAutoplay: fromAutoplay, err: err}
		}
		return nil
	}
}

func buildCurrentQuestion(ctx context.Context, svc *quiz.Service, runtime *session.Runtime, recent []int64) (quiz.Question, error) {
	item, ok := runtime.CurrentItem()
	if !ok {
		return quiz.Question{}, fmt.Errorf("no pending question")
	}
	return svc.BuildQuestion(ctx, item, runtime.Total(), runtime.Session.AnswerMode, recent)
}
