package postgres

import "context"

// LLMRepo tracks LLM token usage and spend for budget enforcement.
type LLMRepo struct{ db *DB }

// LLM returns the LLM usage repository.
func (db *DB) LLM() *LLMRepo { return &LLMRepo{db: db} }

// RecordUsage logs a single LLM call's token counts and cost.
func (r *LLMRepo) RecordUsage(ctx context.Context, model string, inputTokens, outputTokens int, costUSD float64) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO llm_usage (model, input_tokens, output_tokens, cost_usd) VALUES ($1,$2,$3,$4)`,
		model, inputTokens, outputTokens, costUSD)
	return err
}

// RecordAttempt tracks provider requests, including requests rejected by a
// rate limit. This prevents retries from creating a request burst.
func (r *LLMRepo) RecordAttempt(ctx context.Context, model string) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO llm_usage (model, input_tokens, output_tokens, cost_usd) VALUES ($1,0,0,0)`,
		model+":attempt")
	return err
}

// SpendToday returns today's total LLM spend in USD.
func (r *LLMRepo) SpendToday(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COALESCE(sum(cost_usd),0) FROM llm_usage WHERE day = now()::date`).Scan(&total)
	return total, err
}

// CallsLastHour supports a provider-safe rolling request cap.
func (r *LLMRepo) CallsLastHour(ctx context.Context) (int, error) {
	var total int
	err := r.db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM llm_usage WHERE created_at >= now() - interval '1 hour' AND model LIKE '%:attempt'`).Scan(&total)
	return total, err
}

// UsageSummary is a snapshot of today's LLM consumption for the admin panel.
type UsageSummary struct {
	SpendTodayUSD     float64 `json:"spend_today_usd"`
	CallsToday        int     `json:"calls_today"`
	CallsLastHour     int     `json:"calls_last_hour"`
	InputTokensToday  int64   `json:"input_tokens_today"`
	OutputTokensToday int64   `json:"output_tokens_today"`
}

// TodayUsage aggregates today's spend, request count and token totals. Attempt
// rows (logged as "<model>:attempt") count as requests; token/cost totals come
// from completed calls.
func (r *LLMRepo) TodayUsage(ctx context.Context) (UsageSummary, error) {
	var s UsageSummary
	err := r.db.Pool.QueryRow(ctx, `
		SELECT
		  COALESCE(sum(cost_usd),0),
		  count(*) FILTER (WHERE model LIKE '%:attempt'),
		  COALESCE(sum(input_tokens),0),
		  COALESCE(sum(output_tokens),0)
		FROM llm_usage WHERE day = now()::date`).
		Scan(&s.SpendTodayUSD, &s.CallsToday, &s.InputTokensToday, &s.OutputTokensToday)
	if err != nil {
		return s, err
	}
	s.CallsLastHour, err = r.CallsLastHour(ctx)
	return s, err
}
