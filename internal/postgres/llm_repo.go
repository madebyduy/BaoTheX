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

// SpendToday returns today's total LLM spend in USD.
func (r *LLMRepo) SpendToday(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COALESCE(sum(cost_usd),0) FROM llm_usage WHERE day = now()::date`).Scan(&total)
	return total, err
}
