package httpapi

import (
	"context"
	"time"
)

// hasActivePremium is the single entitlement check used by premium-backed
// features. Keeping it server-side prevents a client from bypassing the UI
// lock by calling a premium endpoint directly.
func (s *Server) hasActivePremium(ctx context.Context, userID int64) (bool, error) {
	subscription, err := s.db.Engagement.Subscription(ctx, userID)
	if err != nil {
		return false, err
	}
	return subscription.Active(time.Now()), nil
}
