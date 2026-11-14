package tools

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"squad/features/bots"
	"strings"
	"syscall"
	"time"
)

const maxFetchBytes = 2 << 20

// The model picks these URLs, and a page it reads can tell it what to fetch, so the address is checked at connect time to cover redirects and DNS that points at private hosts.
var fetchClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second, Control: refuseNonPublicAddress}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

// Tailscale and other CGNAT addresses are common on self-hosted setups and reach private machines too.
var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

var hiddenBlocksPattern = regexp.MustCompile(`(?is)<script\b.*?</script>|<style\b.*?</style>|<noscript\b.*?</noscript>|<svg\b.*?</svg>|<template\b.*?</template>|<!--.*?-->`)
var titlePattern = regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title>`)
var headPattern = regexp.MustCompile(`(?is)<head\b.*?</head>`)
var blockTagPattern = regexp.MustCompile(`(?i)</?(p|div|br|li|ul|ol|h[1-6]|tr|table|section|article|header|footer|blockquote|pre|hr)\b[^>]*>`)
var tagPattern = regexp.MustCompile(`(?s)<[^>]+>`)
var spacesPattern = regexp.MustCompile(`[ \t\x{00a0}]+`)
var blankLinesPattern = regexp.MustCompile(`\n{3,}`)

func registerFetchTool() {
	register("web", Tool{
		Name:        "fetch_url",
		Description: "Fetch a web page and return its readable text. Use it to read a link someone shares or a search result in full.",
		Parameters: objectSchema(map[string]any{
			"url": stringProperty("The full http or https URL."),
		}, "url"),
		Run: runFetchURL,
	})
}

func runFetchURL(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		URL string `json:"url"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	parsedURL, err := url.Parse(args.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", fmt.Errorf("url must start with http:// or https://")
	}

	text, err := fetchPageText(ctx, args.URL)
	if err == nil {
		return text, nil
	}

	// Sites that block plain fetches are often already indexed by Exa, the same fallback the research repo uses.
	var statusErr httpStatusError
	isBlocked := errors.As(err, &statusErr) && (statusErr.StatusCode == 401 || statusErr.StatusCode == 403 || statusErr.StatusCode == 429 || statusErr.StatusCode == 451)
	if isBlocked && isExaConfigured() {
		return getExaContents(ctx, args.URL)
	}

	return "", err
}

func fetchPageText(ctx context.Context, pageURL string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; squad/1.0)")
	request.Header.Set("Accept", "text/html,text/plain,application/json;q=0.9,*/*;q=0.5")

	response, err := fetchClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("error fetching page: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return "", httpStatusError{StatusCode: response.StatusCode, Body: response.Status}
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, maxFetchBytes))
	if err != nil {
		return "", fmt.Errorf("error reading page: %w", err)
	}

	contentType := response.Header.Get("Content-Type")
	if strings.Contains(contentType, "html") {
		return htmlToText(string(bodyBytes)), nil
	}

	if strings.HasPrefix(contentType, "text/") || strings.Contains(contentType, "json") || strings.Contains(contentType, "xml") {
		return string(bodyBytes), nil
	}

	return "", fmt.Errorf("unsupported content type %q", contentType)
}

func refuseNonPublicAddress(network, address string, conn syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}

	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}

	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || sharedAddressSpace.Contains(ip) {
		return fmt.Errorf("refusing to fetch non-public address %s", ip)
	}

	return nil
}

func htmlToText(page string) string {
	title := ""
	if match := titlePattern.FindStringSubmatch(page); match != nil {
		title = strings.TrimSpace(html.UnescapeString(match[1]))
	}

	text := hiddenBlocksPattern.ReplaceAllString(page, "")
	text = headPattern.ReplaceAllString(text, "")
	text = blockTagPattern.ReplaceAllString(text, "\n")
	text = tagPattern.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	text = spacesPattern.ReplaceAllString(text, " ")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = blankLinesPattern.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	text = strings.TrimSpace(text)

	if title != "" {
		return "# " + title + "\n\n" + text
	}

	return text
}
