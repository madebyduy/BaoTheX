package httpapi

import "net/http"

// handleAdminLLMUsage reports today's LLM spend and request counts alongside the
// configured caps, so an admin can see which ceiling (Google quota, hourly cap,
// or daily budget) is being approached.
func (s *Server) handleAdminLLMUsage(w http.ResponseWriter, r *http.Request) {
	usage, err := s.db.LLM().TodayUsage(r.Context())
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"spend_today_usd":     usage.SpendTodayUSD,
		"daily_budget_usd":    s.cfg.LLMDailyBudgetUSD,
		"calls_today":         usage.CallsToday,
		"calls_last_hour":     usage.CallsLastHour,
		"max_calls_per_hour":  s.cfg.LLMMaxCallsPerHour,
		"input_tokens_today":  usage.InputTokensToday,
		"output_tokens_today": usage.OutputTokensToday,
		"model":               s.cfg.LLMModel,
	}, nil)
}
