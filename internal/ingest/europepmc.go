package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"repwire/internal/domain"
)

// EuropePMCFetcher queries the Europe PMC REST API. The source's feed_url holds
// the base query; a date window for the current year is appended.
type EuropePMCFetcher struct {
	client *http.Client
}

// NewEuropePMCFetcher constructs an EuropePMCFetcher.
func NewEuropePMCFetcher(client *http.Client) *EuropePMCFetcher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &EuropePMCFetcher{client: client}
}

// Fetch runs the query and returns research RawItems.
func (f *EuropePMCFetcher) Fetch(ctx context.Context, src *domain.Source) (*FetchResult, error) {
	baseQuery := src.FeedURLOrEmpty()
	if baseQuery == "" {
		return &FetchResult{}, nil
	}
	year := time.Now().Year()
	query := fmt.Sprintf("(%s) AND (FIRST_PDATE:[%d-01-01 TO %d-12-31])", baseQuery, year, year)

	q := url.Values{
		"query":      {query},
		"format":     {"json"},
		"resultType": {"core"},
		"pageSize":   {"50"},
		"cursorMark": {"*"},
	}
	u := "https://www.ebi.ac.uk/europepmc/webservices/rest/search?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, &httpError{status: resp.StatusCode, url: "europepmc"}
	}

	var out pmcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	res := &FetchResult{}
	for _, r := range out.ResultList.Result {
		raw, ok := r.toRawItem(src.DefaultLang)
		if !ok {
			continue
		}
		res.Items = append(res.Items, raw)
	}
	return res, nil
}

type pmcResponse struct {
	ResultList struct {
		Result []pmcResult `json:"result"`
	} `json:"resultList"`
}

type pmcResult struct {
	ID              string `json:"id"`
	Source          string `json:"source"`
	PMID            string `json:"pmid"`
	PMCID           string `json:"pmcid"`
	DOI             string `json:"doi"`
	Title           string `json:"title"`
	AbstractText    string `json:"abstractText"`
	AuthorString    string `json:"authorString"`
	JournalTitle    string `json:"journalTitle"`
	PubYear         string `json:"pubYear"`
	FirstPublicDate string `json:"firstPublicationDate"`
	IsOpenAccess    string `json:"isOpenAccess"`
	PubTypeList     struct {
		PubType []string `json:"pubType"`
	} `json:"pubTypeList"`
	FullTextURLList struct {
		FullTextURL []struct {
			URL           string `json:"url"`
			DocumentStyle string `json:"documentStyle"`
		} `json:"fullTextUrl"`
	} `json:"fullTextUrlList"`
}

func (r pmcResult) toRawItem(lang string) (RawItem, bool) {
	if r.Title == "" {
		return RawItem{}, false
	}
	// Prefer a stable canonical link: DOI > PMID > PMCID.
	var link string
	switch {
	case r.DOI != "":
		link = "https://doi.org/" + r.DOI
	case r.PMID != "":
		link = "https://europepmc.org/article/MED/" + r.PMID
	case r.PMCID != "":
		link = "https://europepmc.org/article/PMC/" + r.PMCID
	default:
		return RawItem{}, false
	}

	paper := &domain.ResearchPaper{
		StudyType:    mapStudyType(r.PubTypeList.PubType),
		IsOpenAccess: strings.EqualFold(r.IsOpenAccess, "y"),
		Authors:      splitAuthors(r.AuthorString),
	}
	if r.DOI != "" {
		paper.DOI = ptr(r.DOI)
	}
	if r.PMID != "" {
		paper.PMID = ptr(r.PMID)
	}
	if r.PMCID != "" {
		paper.PMCID = ptr(r.PMCID)
	}
	if r.JournalTitle != "" {
		paper.Journal = ptr(r.JournalTitle)
	}
	if r.AbstractText != "" {
		paper.Abstract = ptr(stripHTML(r.AbstractText))
	}
	if y, err := strconv.Atoi(r.PubYear); err == nil {
		paper.PublishedYear = &y
	}
	if len(r.FullTextURLList.FullTextURL) > 0 {
		paper.FullTextURL = ptr(r.FullTextURLList.FullTextURL[0].URL)
	}

	raw := RawItem{
		Type:     domain.ContentResearch,
		Title:    strings.TrimSpace(r.Title),
		URL:      link,
		Language: lang,
		Research: paper,
	}
	if r.AbstractText != "" {
		abstract := stripHTML(r.AbstractText)
		raw.Excerpt = ptr(truncate(abstract, 200))
		raw.Body = ptr(abstract)
	}
	if t := parsePMCDate(r.FirstPublicDate); t != nil {
		raw.Published = t
	}
	return raw, true
}

// mapStudyType maps Europe PMC pubType strings to our study_type enum.
func mapStudyType(pubTypes []string) domain.StudyType {
	for _, pt := range pubTypes {
		switch strings.ToLower(pt) {
		case "meta-analysis":
			return domain.MetaAnalysis
		case "systematic review":
			return domain.SystematicReview
		case "randomized controlled trial", "randomised controlled trial":
			return domain.RCT
		case "review":
			return domain.NarrativeReview
		case "case reports":
			return domain.CaseStudy
		}
	}
	return domain.StudyOther
}

func splitAuthors(s string) []string {
	s = strings.TrimSpace(strings.TrimSuffix(s, "."))
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func parsePMCDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}
	return nil
}
