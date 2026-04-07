package keymap

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"github.com/harumiWeb/eitango/internal/i18n"
)

type Context string

const (
	ContextGlobal          Context = "global"
	ContextHome            Context = "home"
	ContextHomeConfirm     Context = "home_confirm"
	ContextSettingsOverlay Context = "settings_overlay"
	ContextQuizChoice      Context = "quiz.choice"
	ContextQuizWrite       Context = "quiz.write"
	ContextFeedbackRate    Context = "feedback.rate"
	ContextFeedbackWrite   Context = "feedback.write"
	ContextResults         Context = "results"
	ContextStats           Context = "stats"
	ContextHelp            Context = "help"
)

type Action string

const (
	ActionUp               Action = "up"
	ActionDown             Action = "down"
	ActionLeft             Action = "left"
	ActionRight            Action = "right"
	ActionSelect1          Action = "select1"
	ActionSelect2          Action = "select2"
	ActionSelect3          Action = "select3"
	ActionSelect4          Action = "select4"
	ActionToggleAnswerMode Action = "toggle_answer_mode"
	ActionConfirm          Action = "confirm"
	ActionQuit             Action = "quit"
	ActionHelp             Action = "help"
	ActionSpeak            Action = "speak"
	ActionHint             Action = "hint"
	ActionSkip             Action = "skip"
	ActionToggleAutoplay   Action = "toggle_autoplay"
	ActionWriteQuit        Action = "write_quit"
	ActionAgain            Action = "again"
	ActionHard             Action = "hard"
	ActionGood             Action = "good"
	ActionEasy             Action = "easy"
	ActionNewSession       Action = "new_session"
	ActionReview           Action = "review"
	ActionStats            Action = "stats"
	ActionSettings         Action = "settings"
	ActionBack             Action = "back"
)

type Config struct {
	Version         int                 `toml:"version"`
	Global          map[string][]string `toml:"global"`
	Home            map[string][]string `toml:"home"`
	HomeConfirm     map[string][]string `toml:"home_confirm"`
	SettingsOverlay map[string][]string `toml:"settings_overlay"`
	Quiz            QuizConfig          `toml:"quiz"`
	Feedback        FeedbackConfig      `toml:"feedback"`
	Results         map[string][]string `toml:"results"`
	Stats           map[string][]string `toml:"stats"`
	Help            map[string][]string `toml:"help"`
}

type QuizConfig struct {
	Choice map[string][]string `toml:"choice"`
	Write  map[string][]string `toml:"write"`
}

type FeedbackConfig struct {
	Rate  map[string][]string `toml:"rate"`
	Write map[string][]string `toml:"write"`
}

type State struct {
	values map[Context]map[Action][]string
}

type Conflict struct {
	Context Context
	Action  Action
	Key     string
}

type actionMeta struct {
	labelKey string
}

var contextOrder = []Context{
	ContextHome,
	ContextHomeConfirm,
	ContextSettingsOverlay,
	ContextQuizChoice,
	ContextQuizWrite,
	ContextFeedbackRate,
	ContextFeedbackWrite,
	ContextResults,
	ContextStats,
	ContextHelp,
}

var contextLabels = map[Context]string{
	ContextHome:            "home",
	ContextHomeConfirm:     "home_confirm",
	ContextSettingsOverlay: "settings_overlay",
	ContextQuizChoice:      "quiz.choice",
	ContextQuizWrite:       "quiz.write",
	ContextFeedbackRate:    "feedback.rate",
	ContextFeedbackWrite:   "feedback.write",
	ContextResults:         "results",
	ContextStats:           "stats",
	ContextHelp:            "help",
}

var actionOrder = map[Context][]Action{
	ContextHome: {
		ActionToggleAnswerMode,
		ActionConfirm,
		ActionNewSession,
		ActionReview,
		ActionStats,
		ActionSettings,
		ActionHelp,
		ActionQuit,
	},
	ContextHomeConfirm: {
		ActionConfirm,
		ActionBack,
		ActionHelp,
		ActionQuit,
	},
	ContextSettingsOverlay: {
		ActionUp,
		ActionDown,
		ActionLeft,
		ActionRight,
		ActionConfirm,
		ActionBack,
		ActionHelp,
		ActionQuit,
	},
	ContextQuizChoice: {
		ActionUp,
		ActionDown,
		ActionSelect1,
		ActionSelect2,
		ActionSelect3,
		ActionSelect4,
		ActionConfirm,
		ActionSpeak,
		ActionToggleAutoplay,
		ActionHelp,
		ActionQuit,
	},
	ContextQuizWrite: {
		ActionHint,
		ActionSkip,
		ActionConfirm,
		ActionWriteQuit,
		ActionQuit,
		ActionHelp,
	},
	ContextFeedbackRate: {
		ActionAgain,
		ActionHard,
		ActionGood,
		ActionEasy,
		ActionSpeak,
		ActionToggleAutoplay,
		ActionHelp,
		ActionQuit,
	},
	ContextFeedbackWrite: {
		ActionConfirm,
		ActionSpeak,
		ActionToggleAutoplay,
		ActionHelp,
		ActionQuit,
	},
	ContextResults: {
		ActionConfirm,
		ActionBack,
		ActionHelp,
		ActionQuit,
	},
	ContextStats: {
		ActionConfirm,
		ActionBack,
		ActionHelp,
		ActionQuit,
	},
	ContextHelp: {
		ActionBack,
		ActionQuit,
	},
}

var defaults = map[Context]map[Action][]string{
	ContextHome: {
		ActionToggleAnswerMode: {"tab"},
		ActionConfirm:          {"enter"},
		ActionNewSession:       {"n"},
		ActionReview:           {"r"},
		ActionStats:            {"s"},
		ActionSettings:         {"c"},
		ActionHelp:             {"?"},
		ActionQuit:             {"q", "ctrl+c"},
	},
	ContextHomeConfirm: {
		ActionConfirm: {"enter"},
		ActionBack:    {"esc", "b"},
		ActionHelp:    {"?"},
		ActionQuit:    {"q", "ctrl+c"},
	},
	ContextSettingsOverlay: {
		ActionUp:      {"up", "k"},
		ActionDown:    {"down", "j"},
		ActionLeft:    {"left", "h"},
		ActionRight:   {"right", "l"},
		ActionConfirm: {"enter"},
		ActionBack:    {"esc", "b"},
		ActionHelp:    {"?"},
		ActionQuit:    {"q", "ctrl+c"},
	},
	ContextQuizChoice: {
		ActionUp:             {"up", "k"},
		ActionDown:           {"down", "j"},
		ActionSelect1:        {"1"},
		ActionSelect2:        {"2"},
		ActionSelect3:        {"3"},
		ActionSelect4:        {"4"},
		ActionConfirm:        {"enter"},
		ActionSpeak:          {"ctrl+p"},
		ActionToggleAutoplay: {"shift+tab"},
		ActionHelp:           {"?"},
		ActionQuit:           {"q", "ctrl+c"},
	},
	ContextQuizWrite: {
		ActionHint:      {"tab"},
		ActionSkip:      {"ctrl+s"},
		ActionConfirm:   {"enter"},
		ActionWriteQuit: {"esc"},
		ActionQuit:      {"ctrl+c"},
		ActionHelp:      {"?"},
	},
	ContextFeedbackRate: {
		ActionAgain:          {"a"},
		ActionHard:           {"h"},
		ActionGood:           {"g"},
		ActionEasy:           {"e"},
		ActionSpeak:          {"ctrl+p"},
		ActionToggleAutoplay: {"shift+tab"},
		ActionHelp:           {"?"},
		ActionQuit:           {"q", "ctrl+c"},
	},
	ContextFeedbackWrite: {
		ActionConfirm:        {"enter"},
		ActionSpeak:          {"ctrl+p"},
		ActionToggleAutoplay: {"shift+tab"},
		ActionHelp:           {"?"},
		ActionQuit:           {"q", "ctrl+c"},
	},
	ContextResults: {
		ActionConfirm: {"enter"},
		ActionBack:    {"esc", "b"},
		ActionHelp:    {"?"},
		ActionQuit:    {"q", "ctrl+c"},
	},
	ContextStats: {
		ActionConfirm: {"enter"},
		ActionBack:    {"esc", "b"},
		ActionHelp:    {"?"},
		ActionQuit:    {"q", "ctrl+c"},
	},
	ContextHelp: {
		ActionBack: {"esc", "b"},
		ActionQuit: {"q", "ctrl+c"},
	},
}

var actions = map[Action]actionMeta{
	ActionUp:               {labelKey: i18n.KeyUp},
	ActionDown:             {labelKey: i18n.KeyDown},
	ActionLeft:             {labelKey: i18n.KeyLeft},
	ActionRight:            {labelKey: i18n.KeyRight},
	ActionSelect1:          {labelKey: i18n.KeyChoice1},
	ActionSelect2:          {labelKey: i18n.KeyChoice2},
	ActionSelect3:          {labelKey: i18n.KeyChoice3},
	ActionSelect4:          {labelKey: i18n.KeyChoice4},
	ActionToggleAnswerMode: {labelKey: i18n.KeyToggleMode},
	ActionConfirm:          {labelKey: i18n.KeyConfirm},
	ActionQuit:             {labelKey: i18n.KeyQuit},
	ActionHelp:             {labelKey: i18n.KeyHelp},
	ActionSpeak:            {labelKey: i18n.KeySpeak},
	ActionHint:             {labelKey: i18n.KeyHint},
	ActionSkip:             {labelKey: i18n.KeySkip},
	ActionToggleAutoplay:   {labelKey: i18n.KeyToggleAuto},
	ActionWriteQuit:        {labelKey: i18n.KeyQuit},
	ActionAgain:            {labelKey: i18n.KeyAgain},
	ActionHard:             {labelKey: i18n.KeyHard},
	ActionGood:             {labelKey: i18n.KeyGood},
	ActionEasy:             {labelKey: i18n.KeyEasy},
	ActionNewSession:       {labelKey: i18n.KeyNewSession},
	ActionReview:           {labelKey: i18n.KeyReview},
	ActionStats:            {labelKey: i18n.KeyStats},
	ActionSettings:         {labelKey: i18n.KeySettings},
	ActionBack:             {labelKey: i18n.KeyBack},
}

var namedKeys = map[string]struct{}{
	"up": {}, "down": {}, "left": {}, "right": {}, "tab": {}, "enter": {}, "esc": {},
	"space": {}, "backspace": {}, "delete": {}, "insert": {}, "home": {}, "end": {},
	"pgup": {}, "pgdown": {}, "begin": {}, "find": {}, "return": {},
}

var modifiers = []string{"ctrl", "alt", "shift", "meta", "hyper", "super"}

func Contexts() []Context {
	return append([]Context(nil), contextOrder...)
}

func ContextLabel(ctx Context) string {
	if label, ok := contextLabels[ctx]; ok {
		return label
	}
	return string(ctx)
}

func ActionsForContext(ctx Context) []Action {
	return append([]Action(nil), actionOrder[ctx]...)
}

func ActionLabel(action Action) string {
	if meta, ok := actions[action]; ok {
		return i18n.T(meta.labelKey)
	}
	return string(action)
}

func DefaultKeys(ctx Context, action Action) []string {
	return append([]string(nil), defaults[ctx][action]...)
}

func DefaultState() State {
	values := make(map[Context]map[Action][]string, len(contextOrder))
	for _, ctx := range contextOrder {
		values[ctx] = make(map[Action][]string, len(actionOrder[ctx]))
		for _, action := range actionOrder[ctx] {
			values[ctx][action] = DefaultKeys(ctx, action)
		}
	}
	return State{values: values}
}

func Resolve(cfg Config) (State, error) {
	if cfg.Version != 0 && cfg.Version != 1 {
		return State{}, fmt.Errorf("keymap.version must be 1")
	}

	state := DefaultState()

	if err := applySection(state.values, ContextGlobal, cfg.Global); err != nil {
		return State{}, err
	}
	for _, ctx := range contextOrder {
		for _, action := range actionOrder[ctx] {
			if keys, ok := state.values[ContextGlobal][action]; ok {
				state.values[ctx][action] = append([]string(nil), keys...)
			}
		}
	}
	delete(state.values, ContextGlobal)

	for _, item := range []struct {
		context Context
		table   map[string][]string
	}{
		{ContextHome, cfg.Home},
		{ContextHomeConfirm, cfg.HomeConfirm},
		{ContextSettingsOverlay, cfg.SettingsOverlay},
		{ContextQuizChoice, cfg.Quiz.Choice},
		{ContextQuizWrite, cfg.Quiz.Write},
		{ContextFeedbackRate, cfg.Feedback.Rate},
		{ContextFeedbackWrite, cfg.Feedback.Write},
		{ContextResults, cfg.Results},
		{ContextStats, cfg.Stats},
		{ContextHelp, cfg.Help},
	} {
		if err := applySection(state.values, item.context, item.table); err != nil {
			return State{}, err
		}
	}

	for _, ctx := range contextOrder {
		if err := validateContext(ctx, state.values[ctx]); err != nil {
			return State{}, err
		}
	}

	return state, nil
}

func (s State) Clone() State {
	values := make(map[Context]map[Action][]string, len(s.values))
	for ctx, table := range s.values {
		values[ctx] = make(map[Action][]string, len(table))
		for action, keys := range table {
			values[ctx][action] = append([]string(nil), keys...)
		}
	}
	return State{values: values}
}

func (s State) Keys(ctx Context, action Action) []string {
	if s.values == nil {
		return nil
	}
	return append([]string(nil), s.values[ctx][action]...)
}

func (s *State) SetKeys(ctx Context, action Action, keys []string) error {
	if _, ok := actionOrder[ctx]; !ok {
		return fmt.Errorf("unknown keymap context %q", ctx)
	}
	if !supportsAction(ctx, action) {
		return fmt.Errorf("keymap.%s.%s: unknown action", ContextLabel(ctx), action)
	}
	normalized := make([]string, 0, len(keys))
	for _, raw := range keys {
		token, err := normalizeToken(raw)
		if err != nil {
			return fmt.Errorf("keymap.%s.%s: %w", ContextLabel(ctx), action, err)
		}
		normalized = append(normalized, token)
	}
	next := s.Clone()
	next.ensureContext(ctx)
	next.values[ctx][action] = normalized
	if err := validateContext(ctx, next.values[ctx]); err != nil {
		return err
	}
	s.values = next.values
	return nil
}

func (s State) Match(ctx Context, action Action, msg fmt.Stringer) bool {
	return key.Matches(msg, s.Binding(ctx, action))
}

func (s State) Binding(ctx Context, action Action) key.Binding {
	keys := s.Keys(ctx, action)
	if len(keys) == 0 {
		return key.NewBinding()
	}
	return key.NewBinding(
		key.WithKeys(keys...),
		key.WithHelp(FormatKeys(keys), ActionLabel(action)),
	)
}

func (s State) ConflictsFor(ctx Context, action Action, token string) []Conflict {
	conflicts := make([]Conflict, 0)
	for _, other := range actionOrder[ctx] {
		if other == action {
			continue
		}
		for _, existing := range s.values[ctx][other] {
			if existing == token {
				conflicts = append(conflicts, Conflict{Context: ctx, Action: other, Key: token})
				break
			}
		}
	}
	return conflicts
}

func (s *State) RemoveKey(ctx Context, action Action, token string) error {
	keys := s.Keys(ctx, action)
	next := make([]string, 0, len(keys))
	for _, existing := range keys {
		if existing != token {
			next = append(next, existing)
		}
	}
	return s.SetKeys(ctx, action, next)
}

func (s *State) ReplaceKey(ctx Context, action Action, token string, conflicts []Conflict) error {
	if _, ok := actionOrder[ctx]; !ok {
		return fmt.Errorf("unknown keymap context %q", ctx)
	}
	if !supportsAction(ctx, action) {
		return fmt.Errorf("keymap.%s.%s: unknown action", ContextLabel(ctx), action)
	}

	next := s.Clone()
	next.ensureContext(ctx)

	keys := next.values[ctx][action]
	if !slices.Contains(keys, token) {
		keys = append(keys, token)
	}
	next.values[ctx][action] = keys

	for _, conflict := range conflicts {
		conflictKeys := next.values[conflict.Context][conflict.Action]
		filtered := conflictKeys[:0]
		for _, existing := range conflictKeys {
			if existing != token {
				filtered = append(filtered, existing)
			}
		}
		next.values[conflict.Context][conflict.Action] = filtered
	}

	if err := validateContext(ctx, next.values[ctx]); err != nil {
		return err
	}
	s.values = next.values
	return nil
}

func (s State) ToConfig() Config {
	cfg := Config{}
	for _, ctx := range contextOrder {
		table := make(map[string][]string)
		for _, action := range actionOrder[ctx] {
			keys := s.values[ctx][action]
			if slices.Equal(keys, defaults[ctx][action]) {
				continue
			}
			table[string(action)] = append([]string(nil), keys...)
		}
		if len(table) == 0 {
			continue
		}
		switch ctx {
		case ContextHome:
			cfg.Home = table
		case ContextHomeConfirm:
			cfg.HomeConfirm = table
		case ContextSettingsOverlay:
			cfg.SettingsOverlay = table
		case ContextQuizChoice:
			cfg.Quiz.Choice = table
		case ContextQuizWrite:
			cfg.Quiz.Write = table
		case ContextFeedbackRate:
			cfg.Feedback.Rate = table
		case ContextFeedbackWrite:
			cfg.Feedback.Write = table
		case ContextResults:
			cfg.Results = table
		case ContextStats:
			cfg.Stats = table
		case ContextHelp:
			cfg.Help = table
		}
	}
	if cfg.HasOverrides() {
		cfg.Version = 1
	}
	return cfg
}

func (c Config) HasOverrides() bool {
	return len(c.Global) > 0 ||
		len(c.Home) > 0 ||
		len(c.HomeConfirm) > 0 ||
		len(c.SettingsOverlay) > 0 ||
		len(c.Quiz.Choice) > 0 ||
		len(c.Quiz.Write) > 0 ||
		len(c.Feedback.Rate) > 0 ||
		len(c.Feedback.Write) > 0 ||
		len(c.Results) > 0 ||
		len(c.Stats) > 0 ||
		len(c.Help) > 0
}

func FormatKeys(keys []string) string {
	display := make([]string, 0, len(keys))
	for _, raw := range keys {
		display = append(display, humanizeToken(raw))
	}
	return strings.Join(display, "/")
}

func NormalizeRecordedKey(token string) (string, error) {
	return normalizeToken(token)
}

func IsPrintableCommand(token string) bool {
	if utf8.RuneCountInString(token) != 1 {
		return false
	}
	r, _ := utf8.DecodeRuneInString(token)
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

func applySection(values map[Context]map[Action][]string, ctx Context, table map[string][]string) error {
	if len(table) == 0 {
		return nil
	}
	if ctx != ContextGlobal {
		if _, ok := actionOrder[ctx]; !ok {
			return fmt.Errorf("unknown keymap context %q", ctx)
		}
	}
	if values[ctx] == nil {
		values[ctx] = map[Action][]string{}
	}
	for rawAction, keys := range table {
		action := Action(strings.TrimSpace(rawAction))
		if ctx != ContextGlobal && !supportsAction(ctx, action) {
			return fmt.Errorf("keymap.%s.%s: unknown action", ContextLabel(ctx), rawAction)
		}
		if ctx == ContextGlobal {
			if _, ok := actions[action]; !ok {
				return fmt.Errorf("keymap.global.%s: unknown action", rawAction)
			}
		}
		normalized := make([]string, 0, len(keys))
		seen := map[string]struct{}{}
		for _, raw := range keys {
			token, err := normalizeToken(raw)
			if err != nil {
				return fmt.Errorf("keymap.%s.%s: %w", ContextLabel(ctx), action, err)
			}
			if _, ok := seen[token]; ok {
				return fmt.Errorf("keymap.%s.%s: duplicate key %q", ContextLabel(ctx), action, token)
			}
			seen[token] = struct{}{}
			normalized = append(normalized, token)
		}
		values[ctx][action] = normalized
	}
	return nil
}

func validateContext(ctx Context, table map[Action][]string) error {
	seen := map[string]Action{}
	for _, action := range actionOrder[ctx] {
		keys := table[action]
		perAction := map[string]struct{}{}
		for _, token := range keys {
			if _, ok := perAction[token]; ok {
				return fmt.Errorf("keymap.%s.%s: duplicate key %q", ContextLabel(ctx), action, token)
			}
			perAction[token] = struct{}{}
			if ctx == ContextQuizWrite && IsPrintableCommand(token) {
				return fmt.Errorf("keymap.%s.%s: printable key %q is not allowed in write input", ContextLabel(ctx), action, token)
			}
			if other, ok := seen[token]; ok {
				return fmt.Errorf("keymap.%s.%s: conflicts with keymap.%s.%s (%q)", ContextLabel(ctx), action, ContextLabel(ctx), other, token)
			}
			seen[token] = action
		}
	}
	if ctx == ContextHelp && len(table[ActionBack]) == 0 {
		return fmt.Errorf("keymap.%s.%s: at least one binding must remain", ContextLabel(ctx), ActionBack)
	}
	return nil
}

func normalizeToken(raw string) (string, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return "", fmt.Errorf("key must not be empty")
	}
	if utf8.RuneCountInString(token) == 1 {
		r, _ := utf8.DecodeRuneInString(token)
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", fmt.Errorf("invalid key token %q", raw)
		}
		return token, nil
	}

	parts := strings.Split(strings.ToLower(token), "+")
	if len(parts) == 1 {
		if _, ok := namedKeys[parts[0]]; ok {
			return parts[0], nil
		}
		if isFunctionKey(parts[0]) {
			return parts[0], nil
		}
		return "", fmt.Errorf("invalid key token %q", raw)
	}
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid key token %q", raw)
	}

	base := parts[len(parts)-1]
	if base == "" {
		return "", fmt.Errorf("invalid key token %q", raw)
	}
	if _, ok := namedKeys[base]; !ok && !isFunctionKey(base) && utf8.RuneCountInString(base) != 1 {
		return "", fmt.Errorf("invalid key token %q", raw)
	}

	expectedIndex := 0
	seenMods := map[string]struct{}{}
	for _, mod := range parts[:len(parts)-1] {
		if mod == "" {
			return "", fmt.Errorf("invalid key token %q", raw)
		}
		index := slices.Index(modifiers, mod)
		if index < 0 {
			return "", fmt.Errorf("invalid key token %q", raw)
		}
		if index < expectedIndex {
			return "", fmt.Errorf("invalid key token %q", raw)
		}
		expectedIndex = index
		if _, ok := seenMods[mod]; ok {
			return "", fmt.Errorf("invalid key token %q", raw)
		}
		seenMods[mod] = struct{}{}
	}

	return strings.Join(parts, "+"), nil
}

func supportsAction(ctx Context, action Action) bool {
	return slices.Contains(actionOrder[ctx], action)
}

func (s *State) ensureContext(ctx Context) {
	if s.values == nil {
		s.values = map[Context]map[Action][]string{}
	}
	if s.values[ctx] == nil {
		s.values[ctx] = map[Action][]string{}
	}
}

func humanizeToken(token string) string {
	switch token {
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	case "ctrl+c":
		return "Ctrl+C"
	case "ctrl+p":
		return "Ctrl+P"
	case "ctrl+s":
		return "Ctrl+S"
	case "shift+tab":
		return "Shift+Tab"
	case "tab":
		return "Tab"
	case "enter":
		return "Enter"
	case "esc":
		return "Esc"
	case "backspace":
		return "Backspace"
	case "space":
		return "Space"
	}
	if strings.Contains(token, "+") {
		parts := strings.Split(token, "+")
		for i, part := range parts {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
		return strings.Join(parts, "+")
	}
	if token == "" {
		return ""
	}
	return token
}

func isFunctionKey(token string) bool {
	if !strings.HasPrefix(token, "f") {
		return false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(token, "f"))
	return err == nil && n >= 1 && n <= 63
}
