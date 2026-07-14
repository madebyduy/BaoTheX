package httpapi

import "net/http"

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r.Context())
	page, perPage, _ := pagination(r)
	items, err := s.ranker.PersonalFeed(r.Context(), u.ID, page, perPage)
	if err != nil {
		writeDomainError(w, s.log, err)
		return
	}
	writeJSON(w, http.StatusOK, nonNilItems(items), &Meta{Page: page, PerPage: perPage, Total: len(items)})
}
