package search

import (
	"net/http"
	"squad/commons/utils"
	"strings"
)

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := buildMatchQuery(r.URL.Query().Get("query"))
	if query == "" {
		utils.SendJSONResponse(w, http.StatusOK, []Result{})
		return
	}

	results, err := SearchMessages(query)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "SEARCH_FAILED", "Search failed", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, results)
}

// buildMatchQuery quotes every word so punctuation in what people type can't break FTS5 syntax, and prefix-matches the last word for search as you type.
func buildMatchQuery(input string) string {
	words := strings.Fields(input)
	terms := []string{}

	for i, word := range words {
		term := `"` + strings.ReplaceAll(word, `"`, `""`) + `"`
		if i == len(words)-1 {
			term += "*"
		}
		terms = append(terms, term)
	}

	return strings.Join(terms, " ")
}
