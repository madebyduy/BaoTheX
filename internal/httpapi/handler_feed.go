package httpapi

import (
	"net/http"

	"repwire/internal/domain"
)

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	page, perPage, _ := pagination(r)
	var itemsErr error
	var items = []domain.ContentItem{}
	if r.URL.Query().Get("strict") == "1" {
		premium, err := s.hasActivePremium(r.Context(), u.ID)
		if err != nil {
			writeDomainError(w, s.log, err)
			return
		}
		if premium {
			items, itemsErr = s.ranker.FollowingFeed(r.Context(), u.ID, page, perPage)
		} else {
			// An expired subscription degrades to the balanced personal feed
			// instead of leaving the homepage empty or exposing a stale lock.
			items, itemsErr = s.ranker.PersonalFeed(r.Context(), u.ID, page, perPage)
		}
	} else {
		items, itemsErr = s.ranker.PersonalFeed(r.Context(), u.ID, page, perPage)
	}
	if itemsErr != nil {
		writeDomainError(w, s.log, itemsErr)
		return
	}
	writeJSON(w, http.StatusOK, nonNilItems(items), &Meta{Page: page, PerPage: perPage, Total: len(items)})
}
