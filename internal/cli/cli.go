package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/crevas/Apple-Ads-CLI/internal/appleads"
	"github.com/crevas/Apple-Ads-CLI/internal/auth"
	"github.com/crevas/Apple-Ads-CLI/internal/config"
	"github.com/crevas/Apple-Ads-CLI/internal/lilycloud"
	"github.com/crevas/Apple-Ads-CLI/internal/output"
	"github.com/crevas/Apple-Ads-CLI/internal/providers/campaignv5"
	"github.com/crevas/Apple-Ads-CLI/internal/providers/platform"
)

const version = "0.2.1"

type globalOptions struct {
	Provider string
	Output   string
	Verbose  bool
}

func Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printHelp(stdout)
		return 0
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Fprintf(stdout, "lily %s\n", version)
		return 0
	}

	globals, rest, err := parseGlobal(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if len(rest) == 0 {
		printHelp(stdout)
		return 0
	}

	switch rest[0] {
	case "login":
		return runLogin(rest[1:], stdout, stderr)
	case "logout":
		return runLogout(stdout, stderr)
	case "auth":
		return runAuth(rest[1:], stdout, stderr)
	case "ads":
		return runAds(ctx, rest[1:], globals, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", rest[0])
		printHelp(stderr)
		return 2
	}
}

func parseGlobal(args []string) (globalOptions, []string, error) {
	opts := globalOptions{Output: "json"}
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--provider":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--provider requires a value")
			}
			opts.Provider = args[i]
		case "--output", "-o":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--output requires a value")
			}
			opts.Output = args[i]
		case "--verbose", "-v":
			opts.Verbose = true
		default:
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

func runLogin(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lily login", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var token string
	flags.StringVar(&token, "token", "", "Lily API token")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if strings.TrimSpace(token) == "" {
		return printValue(stdout, "json", map[string]any{
			"source": lilycloud.ProductName,
			"status": "token_required",
			"nextActions": []string{
				"Optional: create a Lily CLI token in Lily Ads Revenue Analytics if you want keyword-level revenue and ROAS enrichment.",
				"`lily login --token <token>` is not required for Apple Ads campaign planning or Apple Ads API operations.",
				"To manage Apple Ads, configure Apple Ads API credentials locally and run `lily ads doctor`. Private keys stay on this machine.",
			},
		}, stderr)
	}
	if err := config.SaveLilyToken(token); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return printValue(stdout, "json", map[string]any{
		"source":  lilycloud.ProductName,
		"status":  "ok",
		"message": "Lily login saved.",
	}, stderr)
}

func runLogout(stdout io.Writer, stderr io.Writer) int {
	if err := config.ClearLilyToken(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return printValue(stdout, "json", map[string]any{
		"source":  lilycloud.ProductName,
		"status":  "ok",
		"message": "Lily login removed.",
	}, stderr)
}

func runAuth(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout, "Usage:", "  lily auth status")
		return 0
	}
	if args[0] != "status" {
		fmt.Fprintf(stderr, "unknown auth command %q\n", args[0])
		return 2
	}
	return printValue(stdout, "json", lilycloud.New(config.Load()).AuthStatus(), stderr)
}

func runAds(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printAdsHelp(stdout)
		return 0
	}
	switch args[0] {
	case "doctor":
		return runDoctor(globals, stdout, stderr)
	case "platform":
		return runPlatform(args[1:], stdout, stderr)
	case "plan":
		return runPlan(ctx, args[1:], globals, stdout, stderr)
	case "reports":
		return runReports(ctx, args[1:], globals, stdout, stderr)
	case "revenue":
		return runRevenue(args[1:], stdout, stderr)
	case "suggestions":
		return runSuggestions(ctx, args[1:], globals, stdout, stderr)
	case "recommendations":
		return runRecommendations(ctx, args[1:], globals, stdout, stderr)
	case "insights":
		return runInsights(ctx, args[1:], globals, stdout, stderr)
	case "change-history":
		return runChangeHistory(ctx, args[1:], globals, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown ads command %q\n\n", args[0])
		printAdsHelp(stderr)
		return 2
	}
}

func runDoctor(globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	cfg := config.Load()
	if globals.Provider != "" {
		cfg.Provider = config.NormalizeProvider(globals.Provider)
	}
	authErr := cfg.ValidateAuth()
	scopeErr := cfg.ValidateProviderScope()
	lilyLoggedIn := strings.TrimSpace(cfg.LilyToken) != ""
	appleAdsCredentials := map[string]any{
		"configured":         authErr == nil,
		"storage":            "local_environment_or_config_file",
		"configPath":         config.ConfigPath(),
		"privateKeyUploaded": false,
		"requiredFor": []string{
			"Apple Ads API calls",
			"executed campaign changes",
			"Apple Ads reporting",
		},
	}
	if authErr != nil {
		appleAdsCredentials["error"] = authErr.Error()
	}
	providerScope := map[string]any{
		"configured": scopeErr == nil,
		"provider":   cfg.Provider,
	}
	if scopeErr != nil {
		providerScope["error"] = scopeErr.Error()
	}

	checks := map[string]any{
		"tool":                  "Apple Ads CLI by Lily",
		"version":               version,
		"provider":              cfg.Provider,
		"configPath":            config.ConfigPath(),
		"appleAdsReady":         authErr == nil && scopeErr == nil,
		"appleAdsCredentials":   appleAdsCredentials,
		"providerScope":         providerScope,
		"v5Base":                cfg.CampaignV5Base,
		"platformBase":          cfg.PlatformBase,
		"revenueAnalyticsReady": lilyLoggedIn,
		"lilyLogin": map[string]any{
			"loggedIn":                      lilyLoggedIn,
			"requiredForAppleAdsOperations": false,
			"optionalFor": []string{
				"keyword-level revenue analytics",
				"paid-user and ROAS enrichment",
				"Lily Ads Revenue Analytics cloud reports",
			},
		},
		"nextSteps": []string{
			"Configure Apple Ads API credentials locally with environment variables or the config file. Private keys stay on this machine.",
			"Run `lily ads doctor` again until appleAdsReady is true.",
			"Optional: run `lily login --token <token>` only when you want Lily Ads Revenue Analytics revenue and ROAS enrichment.",
		},
	}
	return printValue(stdout, globals.Output, checks, stderr)
}

func runPlatform(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout,
			"Usage:",
			"  lily ads platform readiness",
			"",
			"Checks the local CLI build for Platform API readiness. It does not call Apple.",
		)
		return 0
	}
	switch args[0] {
	case "readiness":
		return printValue(stdout, "json", map[string]any{
			"tool":               "Apple Ads CLI by Lily",
			"platformApiReady":   true,
			"defaultProvider":    "campaignv5",
			"supportedProviders": []string{"campaignv5", "platform"},
			"compatibility": map[string]any{
				"auth":                  "shared",
				"v5ContextHeader":       "X-AP-Context: orgId={orgId}",
				"platformContextHeader": "X-AP-Context: adAccountId={adAccountId}",
				"v5ResponseField":       "data",
				"platformResponseField": "result",
				"businessPlanCommand":   "lily ads plan create",
			},
			"notes": []string{
				"Platform provider is available behind --provider platform.",
				"Live Platform API v1 campaign reporting and read-only opportunity queries are supported; campaignv5 remains the default during migration.",
			},
			"readOnlyOpportunityCommands": []string{
				"suggestions keywords",
				"suggestions target-cpa",
				"recommendations keywords",
				"recommendations target-cpa",
				"recommendations daily-budget",
				"insights search-term-popularity",
				"insights impression-share",
				"change-history query",
				"change-history detail",
			},
		}, stderr)
	default:
		fmt.Fprintf(stderr, "unknown ads platform command %q\n", args[0])
		return 2
	}
}

func runReports(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout,
			"Usage:",
			"  lily ads reports campaigns --app-id <adamId> --from YYYY-MM-DD --to YYYY-MM-DD",
			"",
			"Campaign reports default to Lily Ads Revenue Analytics enrichment when Lily login is available.",
		)
		return 0
	}
	if args[0] != "campaigns" {
		fmt.Fprintf(stderr, "unknown ads reports command %q\n", args[0])
		return 2
	}
	return runCampaignReport(ctx, args[1:], globals, stdout, stderr)
}

func runCampaignReport(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lily ads reports campaigns", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var input appleads.CampaignReportQuery
	var noRevenue bool
	flags.StringVar(&input.AppID, "app-id", "", "App Store adamId")
	flags.StringVar(&input.From, "from", "", "start date YYYY-MM-DD")
	flags.StringVar(&input.To, "to", "", "end date YYYY-MM-DD")
	flags.StringVar(&input.TimeZone, "timezone", "ORTZ", "report timezone")
	flags.StringVar(&input.Granularity, "granularity", "DAILY", "DAILY, WEEKLY, or MONTHLY")
	flags.IntVar(&input.Limit, "limit", 100, "max rows")
	flags.IntVar(&input.Offset, "offset", 0, "pagination offset")
	flags.BoolVar(&noRevenue, "no-revenue", false, "skip Lily Ads Revenue Analytics enrichment")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	input = defaultReportRange(input)

	cfg := config.Load()
	if globals.Provider != "" {
		cfg.Provider = config.NormalizeProvider(globals.Provider)
	}
	provider, client, err := buildProvider(ctx, cfg, globals.Verbose, stderr, true)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	appleReport, err := provider.QueryCampaignReport(client, input)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	revenue := lilycloud.RevenueStatus{
		Source: lilycloud.ProductName,
		Status: "skipped",
		Notice: "Revenue enrichment was skipped by --no-revenue.",
	}
	if !noRevenue {
		revenue = lilycloud.New(cfg).RevenueSummary(revenueQueryWithAppleAdsContext(cfg, lilycloud.RevenueQuery{
			AppID: input.AppID,
			From:  input.From,
			To:    input.To,
		}))
	}

	return printValue(stdout, globals.Output, map[string]any{
		"tool":       "Apple Ads CLI by Lily",
		"provider":   provider.Name(),
		"reportType": "campaigns",
		"range": map[string]any{
			"from": input.From,
			"to":   input.To,
		},
		"appleAds": appleReport,
		"revenue":  revenue,
		"roas":     nil,
		"notice":   revenue.Notice,
	}, stderr)
}

func runRevenue(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout, "Usage:", "  lily ads revenue summary --app-id <adamId> --from YYYY-MM-DD --to YYYY-MM-DD")
		return 0
	}
	if args[0] != "summary" {
		fmt.Fprintf(stderr, "unknown ads revenue command %q\n", args[0])
		return 2
	}
	flags := flag.NewFlagSet("lily ads revenue summary", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var query lilycloud.RevenueQuery
	flags.StringVar(&query.AppID, "app-id", "", "App Store adamId")
	flags.StringVar(&query.From, "from", "", "start date YYYY-MM-DD")
	flags.StringVar(&query.To, "to", "", "end date YYYY-MM-DD")
	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	reportRange := defaultReportRange(appleads.CampaignReportQuery{AppID: query.AppID, From: query.From, To: query.To})
	query.From = reportRange.From
	query.To = reportRange.To
	cfg := config.Load()
	query = revenueQueryWithAppleAdsContext(cfg, query)
	status := lilycloud.New(cfg).RevenueSummary(query)
	exitCode := 0
	if status.Status == "login_required" || status.Status == "dashboard_required" || status.Status == "account_mismatch" {
		exitCode = 3
	}
	if err := output.JSON(stdout, status); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return exitCode
}

func revenueQueryWithAppleAdsContext(cfg config.Config, query lilycloud.RevenueQuery) lilycloud.RevenueQuery {
	query.AppleAdsProvider = config.NormalizeProvider(cfg.Provider)
	switch query.AppleAdsProvider {
	case "campaignv5":
		query.AppleAdsOrgID = strings.TrimSpace(cfg.OrgID)
	case "platform":
		query.AppleAdsAdAccountID = strings.TrimSpace(cfg.AdAccountID)
	}
	return query
}

func runSuggestions(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout,
			"Usage:",
			"  lily --provider platform ads suggestions keywords --app-id <adamId> [--terms term1,term2] [--countries US,GB]",
			"  lily --provider platform ads suggestions target-cpa --app-id <adamId>",
		)
		return 0
	}

	suggestionType, ok := normalizeSuggestionType(args[0])
	if !ok {
		fmt.Fprintf(stderr, "unknown ads suggestions command %q\n", args[0])
		return 2
	}
	flags := flag.NewFlagSet("lily ads suggestions "+args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var input appleads.SuggestionQuery
	var terms string
	var countries string
	flags.StringVar(&input.AppID, "app-id", "", "App Store adamId")
	flags.StringVar(&terms, "terms", "", "comma-separated seed terms")
	flags.StringVar(&countries, "countries", "", "comma-separated country or region codes")
	flags.StringVar(&countries, "country", "", "country or region code")
	flags.IntVar(&input.Limit, "limit", 50, "max rows")
	flags.IntVar(&input.Offset, "offset", 0, "pagination offset")
	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	appID, err := normalizeAppID(input.AppID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := validatePagination(input.Limit, input.Offset); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	input.AppID = appID
	input.Type = suggestionType
	input.Terms = splitList(terms, false)
	input.Countries = normalizeCountries(countries)

	return executeOpportunityQuery(ctx, globals, stdout, stderr, "suggestions."+strings.ToLower(suggestionType), input,
		func(provider appleads.OpportunityProvider, client appleads.RequestContext) (appleads.RawResponse, error) {
			return provider.QuerySuggestions(client, input)
		})
}

func runRecommendations(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout,
			"Usage:",
			"  lily --provider platform ads recommendations keywords --app-id <adamId> [--state AVAILABLE]",
			"  lily --provider platform ads recommendations target-cpa --app-id <adamId> [--state AVAILABLE]",
			"  lily --provider platform ads recommendations daily-budget --app-id <adamId> [--state AVAILABLE]",
			"  lily --provider platform ads recommendations query --type keyword|target-cpa|daily-budget --app-id <adamId>",
		)
		return 0
	}
	if args[0] == "apply" {
		return printReserved(stdout, stderr, "recommendations.apply", map[string]any{
			"arguments": args[1:],
		})
	}

	command := args[0]
	flags := flag.NewFlagSet("lily ads recommendations "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var input appleads.RecommendationQuery
	var typeFlag string
	flags.StringVar(&input.AppID, "app-id", "", "App Store adamId")
	flags.StringVar(&typeFlag, "type", "", "keyword, target-cpa, or daily-budget")
	flags.StringVar(&input.State, "state", "AVAILABLE", "AVAILABLE, APPLIED, DISMISSED, or ALL")
	flags.IntVar(&input.Limit, "limit", 50, "max rows")
	flags.IntVar(&input.Offset, "offset", 0, "pagination offset")
	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if command == "query" {
		command = typeFlag
	}
	recommendationType, ok := normalizeRecommendationType(command)
	if !ok {
		fmt.Fprintf(stderr, "unsupported recommendation type %q; use keyword, target-cpa, or daily-budget\n", command)
		return 2
	}
	appID, err := normalizeAppID(input.AppID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := validatePagination(input.Limit, input.Offset); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	input.AppID = appID
	input.Type = recommendationType
	input.State = strings.ToUpper(strings.TrimSpace(input.State))
	if input.State == "ALL" {
		input.State = ""
	}
	if input.State != "" && input.State != "AVAILABLE" && input.State != "APPLIED" && input.State != "DISMISSED" {
		fmt.Fprintln(stderr, "--state must be AVAILABLE, APPLIED, DISMISSED, or ALL")
		return 2
	}

	return executeOpportunityQuery(ctx, globals, stdout, stderr, "recommendations."+strings.ToLower(recommendationType), input,
		func(provider appleads.OpportunityProvider, client appleads.RequestContext) (appleads.RawResponse, error) {
			return provider.QueryRecommendations(client, input)
		})
}

func runInsights(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout,
			"Usage:",
			"  lily --provider platform ads insights search-term-popularity --country US --genre 'Photo & Video' [--granularity weekly|monthly]",
			"  lily --provider platform ads insights impression-share --app-id <adamId> [--countries US,GB] [--search-terms term1,term2]",
		)
		return 0
	}
	switch args[0] {
	case "search-term-popularity", "popularity":
		return runSearchTermPopularity(ctx, args[1:], globals, stdout, stderr)
	case "impression-share":
		return runImpressionShare(ctx, args[1:], globals, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown ads insights command %q\n", args[0])
		return 2
	}
}

func runSearchTermPopularity(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lily ads insights search-term-popularity", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var input appleads.SearchTermPopularityQuery
	flags.StringVar(&input.Country, "country", "", "country or region code")
	flags.StringVar(&input.Genre, "genre", "", "App Store genre, e.g. Photo & Video")
	flags.StringVar(&input.From, "from", "", "start date YYYY-MM-DD")
	flags.StringVar(&input.To, "to", "", "end date YYYY-MM-DD")
	flags.StringVar(&input.Granularity, "granularity", "WEEKLY_SUN_SAT", "weekly or monthly")
	flags.IntVar(&input.Limit, "limit", 100, "max rows")
	flags.IntVar(&input.Offset, "offset", 0, "pagination offset")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	input.Country = strings.ToUpper(strings.TrimSpace(input.Country))
	if input.Country == "UK" {
		input.Country = "GB"
	}
	input.Genre = strings.TrimSpace(input.Genre)
	input.Granularity = normalizePopularityGranularity(input.Granularity)
	if input.Country == "" {
		fmt.Fprintln(stderr, "--country is required")
		return 2
	}
	if input.Genre == "" {
		fmt.Fprintln(stderr, "--genre is required")
		return 2
	}
	if input.Granularity == "" {
		fmt.Fprintln(stderr, "--granularity must be weekly or monthly")
		return 2
	}
	input.From, input.To = defaultPopularityRange(input.From, input.To, input.Granularity, time.Now().UTC())
	if err := validateDateRange(input.From, input.To); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := validatePagination(input.Limit, input.Offset); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	return executeOpportunityQuery(ctx, globals, stdout, stderr, "insights.search-term-popularity", input,
		func(provider appleads.OpportunityProvider, client appleads.RequestContext) (appleads.RawResponse, error) {
			return provider.QuerySearchTermPopularity(client, input)
		})
}

func runImpressionShare(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lily ads insights impression-share", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var input appleads.ImpressionShareQuery
	var countries string
	var searchTerms string
	flags.StringVar(&input.AppID, "app-id", "", "App Store adamId")
	flags.StringVar(&input.From, "from", "", "start date YYYY-MM-DD")
	flags.StringVar(&input.To, "to", "", "end date YYYY-MM-DD")
	flags.StringVar(&input.Granularity, "granularity", "DAILY", "daily, weekly, or monthly")
	flags.StringVar(&countries, "countries", "", "comma-separated country or region codes")
	flags.StringVar(&countries, "country", "", "country or region code")
	flags.StringVar(&searchTerms, "search-terms", "", "comma-separated search terms")
	flags.IntVar(&input.Limit, "limit", 100, "max rows")
	flags.IntVar(&input.Offset, "offset", 0, "pagination offset")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	appID, err := normalizeAppID(input.AppID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	input.AppID = appID
	input.Granularity = normalizeReportGranularity(input.Granularity)
	if input.Granularity == "" {
		fmt.Fprintln(stderr, "--granularity must be daily, weekly, or monthly")
		return 2
	}
	input.From, input.To = defaultCompletedDateRange(input.From, input.To, 7, time.Now().UTC())
	if err := validateDateRange(input.From, input.To); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := validatePagination(input.Limit, input.Offset); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	input.Countries = normalizeCountries(countries)
	input.SearchTerms = splitList(searchTerms, false)

	return executeOpportunityQuery(ctx, globals, stdout, stderr, "insights.impression-share", input,
		func(provider appleads.OpportunityProvider, client appleads.RequestContext) (appleads.RawResponse, error) {
			return provider.QueryImpressionShare(client, input)
		})
}

func runChangeHistory(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout,
			"Usage:",
			"  lily --provider platform ads change-history query [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--entity-types Campaign,Keyword] [--event-types UPDATE]",
			"  lily --provider platform ads change-history detail --id <detailId>",
		)
		return 0
	}
	if args[0] == "detail" {
		flags := flag.NewFlagSet("lily ads change-history detail", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		var detailID string
		flags.StringVar(&detailID, "id", "", "change-history detail id")
		if err := flags.Parse(args[1:]); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		detailID = strings.TrimSpace(detailID)
		if detailID == "" {
			fmt.Fprintln(stderr, "--id is required")
			return 2
		}
		return executeOpportunityQuery(ctx, globals, stdout, stderr, "change-history.detail", map[string]any{"detailId": detailID},
			func(provider appleads.OpportunityProvider, client appleads.RequestContext) (appleads.RawResponse, error) {
				return provider.GetChangeHistoryDetail(client, detailID)
			})
	}
	if args[0] != "query" {
		fmt.Fprintf(stderr, "unknown ads change-history command %q\n", args[0])
		return 2
	}
	flags := flag.NewFlagSet("lily ads change-history query", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var input appleads.ChangeHistoryQuery
	var entityTypes string
	var eventTypes string
	flags.StringVar(&input.From, "from", "", "start date YYYY-MM-DD")
	flags.StringVar(&input.To, "to", "", "end date YYYY-MM-DD")
	flags.StringVar(&entityTypes, "entity-types", "", "comma-separated entity types")
	flags.StringVar(&eventTypes, "event-types", "", "comma-separated CREATE, UPDATE, or DELETE")
	flags.IntVar(&input.Limit, "limit", 50, "max rows")
	flags.IntVar(&input.Offset, "offset", 0, "pagination offset")
	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	input.From, input.To = defaultCompletedDateRange(input.From, input.To, 7, time.Now().UTC())
	if err := validateDateRange(input.From, input.To); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := validatePagination(input.Limit, input.Offset); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	input.EntityTypes = splitList(entityTypes, false)
	if len(input.EntityTypes) == 0 {
		fmt.Fprintln(stderr, "--entity-types is required by the Platform API")
		return 2
	}
	input.EventTypes = splitList(eventTypes, true)
	for _, eventType := range input.EventTypes {
		if eventType != "CREATE" && eventType != "UPDATE" && eventType != "DELETE" {
			fmt.Fprintln(stderr, "--event-types values must be CREATE, UPDATE, or DELETE")
			return 2
		}
	}

	return executeOpportunityQuery(ctx, globals, stdout, stderr, "change-history.query", input,
		func(provider appleads.OpportunityProvider, client appleads.RequestContext) (appleads.RawResponse, error) {
			return provider.QueryChangeHistory(client, input)
		})
}

func executeOpportunityQuery(
	ctx context.Context,
	globals globalOptions,
	stdout io.Writer,
	stderr io.Writer,
	feature string,
	requested any,
	query func(appleads.OpportunityProvider, appleads.RequestContext) (appleads.RawResponse, error),
) int {
	cfg := config.Load()
	if globals.Provider != "" {
		cfg.Provider = config.NormalizeProvider(globals.Provider)
	}
	provider, client, err := buildOpportunityProvider(ctx, cfg, globals.Verbose, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	response, err := query(provider, client)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return printValue(stdout, globals.Output, map[string]any{
		"tool":      "Apple Ads CLI by Lily",
		"provider":  provider.Name(),
		"feature":   feature,
		"requested": requested,
		"appleAds":  response,
	}, stderr)
}

func printReserved(stdout io.Writer, stderr io.Writer, feature string, request map[string]any) int {
	return printValue(stdout, "json", map[string]any{
		"tool":      "Apple Ads CLI by Lily",
		"feature":   feature,
		"status":    "reserved",
		"provider":  "platform",
		"requested": request,
		"message":   "Read-only Platform API recommendation queries are supported. Applying or dismissing recommendations is not enabled until the write contract is verified.",
		"supportedCommands": []string{
			"lily --provider platform ads recommendations keywords --app-id <adamId>",
			"lily --provider platform ads recommendations target-cpa --app-id <adamId>",
			"lily --provider platform ads recommendations daily-budget --app-id <adamId>",
		},
	}, stderr)
}

func normalizeSuggestionType(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "keyword", "keywords":
		return "KEYWORD", true
	case "cpa", "target-cpa", "target-cpas":
		return "TARGET_CPA", true
	default:
		return "", false
	}
}

func normalizeRecommendationType(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "keyword", "keywords":
		return "KEYWORD", true
	case "cpa", "target-cpa", "target-cpas":
		return "TARGET_CPA", true
	case "budget", "daily-budget", "daily-budgets":
		return "DAILY_BUDGET", true
	default:
		return "", false
	}
}

func normalizeAppID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("--app-id is required")
	}
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return "", fmt.Errorf("--app-id must be a numeric App Store adamId")
	}
	return value, nil
}

func splitList(value string, upper bool) []string {
	var result []string
	seen := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if upper {
			item = strings.ToUpper(item)
		}
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func normalizeCountries(value string) []string {
	countries := splitList(value, true)
	for index, country := range countries {
		if country == "UK" {
			countries[index] = "GB"
		}
	}
	return countries
}

func validatePagination(limit int, offset int) error {
	if limit < 1 || limit > 1000 {
		return fmt.Errorf("--limit must be between 1 and 1000")
	}
	if offset < 0 {
		return fmt.Errorf("--offset must be zero or greater")
	}
	return nil
}

func validateDateRange(from string, to string) error {
	const layout = "2006-01-02"
	start, err := time.Parse(layout, from)
	if err != nil {
		return fmt.Errorf("--from must use YYYY-MM-DD")
	}
	end, err := time.Parse(layout, to)
	if err != nil {
		return fmt.Errorf("--to must use YYYY-MM-DD")
	}
	if start.After(end) {
		return fmt.Errorf("--from must not be after --to")
	}
	return nil
}

func normalizePopularityGranularity(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "WEEKLY", "WEEKLY_SUN_SAT":
		return "WEEKLY_SUN_SAT"
	case "MONTH", "MONTHLY":
		return "MONTHLY"
	default:
		return ""
	}
}

func normalizeReportGranularity(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "DAY", "DAILY":
		return "DAILY"
	case "WEEK", "WEEKLY":
		return "WEEKLY"
	case "MONTH", "MONTHLY":
		return "MONTHLY"
	default:
		return ""
	}
}

func defaultPopularityRange(from string, to string, granularity string, now time.Time) (string, string) {
	if from != "" && to != "" {
		return from, to
	}
	utcToday := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	var start time.Time
	var end time.Time
	if granularity == "MONTHLY" {
		start = time.Date(utcToday.Year(), utcToday.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		end = time.Date(utcToday.Year(), utcToday.Month(), 0, 0, 0, 0, 0, time.UTC)
	} else {
		daysSincePreviousSaturday := int(utcToday.Weekday()) + 1
		end = utcToday.AddDate(0, 0, -daysSincePreviousSaturday)
		start = end.AddDate(0, 0, -6)
	}
	if from == "" {
		from = start.Format("2006-01-02")
	}
	if to == "" {
		to = end.Format("2006-01-02")
	}
	return from, to
}

func defaultCompletedDateRange(from string, to string, days int, now time.Time) (string, string) {
	if days < 1 {
		days = 1
	}
	utcToday := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	end := utcToday.AddDate(0, 0, -1)
	start := end.AddDate(0, 0, -days+1)
	if from == "" {
		from = start.Format("2006-01-02")
	}
	if to == "" {
		to = end.Format("2006-01-02")
	}
	return from, to
}

func runPlan(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printPlanHelp(stdout)
		return 0
	}
	command := args[0]
	if command != "create" && command != "recommend" {
		fmt.Fprintf(stderr, "unknown ads plan command %q\n", args[0])
		return 2
	}

	input, err := parsePlanCreate(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if command == "recommend" {
		input.Execute = false
	}

	cfg := config.Load()
	if globals.Provider != "" {
		cfg.Provider = config.NormalizeProvider(globals.Provider)
	}
	input.ProviderName = cfg.Provider
	if input.Currency == "" {
		input.Currency = cfg.DefaultCurrency
	}

	provider, client, err := buildProvider(ctx, cfg, globals.Verbose, stderr, input.Execute)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	service := appleads.PlanService{Provider: provider, Client: client}
	result, err := service.Create(input)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return printValue(stdout, globals.Output, result, stderr)
}

func parsePlanCreate(args []string) (appleads.PlanCreateInput, error) {
	flags := flag.NewFlagSet("lily ads plan create", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var countries string
	var keywords string
	var exactKeywords string
	var broadKeywords string
	var negativeExact string
	var negativeBroad string
	var campaignNegativeExact string
	var campaignNegativeBroad string
	var adGroupNegativeExact string
	var adGroupNegativeBroad string
	var creative string
	var creativeID string
	var productPageID string
	var creativeName string
	var adName string
	var execute bool
	var yes bool
	var dryRun bool
	var input appleads.PlanCreateInput

	flags.StringVar(&input.Name, "name", "", "campaign plan name")
	flags.StringVar(&input.AppID, "app-id", "", "App Store adamId")
	flags.StringVar(&countries, "country", "", "country or region code, e.g. GB")
	flags.StringVar(&countries, "countries", "", "comma-separated country or region codes")
	flags.StringVar(&input.Currency, "currency", "", "currency code")
	flags.StringVar(&input.DailyBudget, "daily-budget", "", "campaign daily budget amount")
	flags.StringVar(&input.AdGroupName, "adgroup", "", "ad group name")
	flags.StringVar(&input.DefaultBid, "bid", "", "default keyword/ad group bid")
	flags.StringVar(&input.CPAGoal, "cpa-goal", "", "optional target CPA / CPA goal")
	flags.StringVar(&keywords, "keywords", "", "comma-separated exact keywords")
	flags.StringVar(&exactKeywords, "exact-keywords", "", "comma-separated exact keywords")
	flags.StringVar(&broadKeywords, "broad-keywords", "", "comma-separated broad keywords")
	flags.StringVar(&negativeExact, "negative-exact", "", "comma-separated ad group exact negative keywords")
	flags.StringVar(&negativeBroad, "negative-broad", "", "comma-separated ad group broad negative keywords")
	flags.StringVar(&campaignNegativeExact, "campaign-negative-exact", "", "comma-separated campaign exact negative keywords")
	flags.StringVar(&campaignNegativeBroad, "campaign-negative-broad", "", "comma-separated campaign broad negative keywords")
	flags.StringVar(&adGroupNegativeExact, "adgroup-negative-exact", "", "comma-separated ad group exact negative keywords")
	flags.StringVar(&adGroupNegativeBroad, "adgroup-negative-broad", "", "comma-separated ad group broad negative keywords")
	flags.StringVar(&creative, "creative", "", "creative mode: none, default, product-page, product-page:<id>, creative-id:<id>")
	flags.StringVar(&creativeID, "creative-id", "", "existing Apple Ads creative id to attach")
	flags.StringVar(&productPageID, "product-page-id", "", "App Store custom product page id for creative")
	flags.StringVar(&creativeName, "creative-name", "", "optional creative name")
	flags.StringVar(&adName, "ad-name", "", "optional ad name")
	flags.StringVar(&input.StartTime, "start-time", "", "optional start time")
	flags.StringVar(&input.EndTime, "end-time", "", "optional end time")
	flags.StringVar(&input.Status, "status", "PAUSED", "initial status")
	flags.StringVar(&input.Supply, "supply", "APPSTORE_SEARCH_RESULTS", "supply placement")
	flags.BoolVar(&input.AllowPartial, "allow-partial", true, "allow partial keyword bulk success when provider supports it")
	flags.BoolVar(&execute, "execute", false, "execute write operations")
	flags.BoolVar(&yes, "yes", false, "confirm write operations")
	flags.BoolVar(&dryRun, "dry-run", false, "force dry-run")
	flags.StringVar(&input.CorrelationID, "correlation-id", "", "optional id for AI-agent traceability")

	if err := flags.Parse(args); err != nil {
		return input, err
	}

	input.Countries = appleads.NormalizeCountries([]string{countries})
	input.Keywords = append(input.Keywords, appleads.ParseKeywords(keywords, "EXACT", input.Currency, input.DefaultBid)...)
	input.Keywords = append(input.Keywords, appleads.ParseKeywords(exactKeywords, "EXACT", input.Currency, input.DefaultBid)...)
	input.Keywords = append(input.Keywords, appleads.ParseKeywords(broadKeywords, "BROAD", input.Currency, input.DefaultBid)...)
	input.CampaignNegativeKeywords = append(input.CampaignNegativeKeywords, appleads.ParseNegativeKeywords(campaignNegativeExact, "EXACT")...)
	input.CampaignNegativeKeywords = append(input.CampaignNegativeKeywords, appleads.ParseNegativeKeywords(campaignNegativeBroad, "BROAD")...)
	input.AdGroupNegativeKeywords = append(input.AdGroupNegativeKeywords, appleads.ParseNegativeKeywords(negativeExact, "EXACT")...)
	input.AdGroupNegativeKeywords = append(input.AdGroupNegativeKeywords, appleads.ParseNegativeKeywords(negativeBroad, "BROAD")...)
	input.AdGroupNegativeKeywords = append(input.AdGroupNegativeKeywords, appleads.ParseNegativeKeywords(adGroupNegativeExact, "EXACT")...)
	input.AdGroupNegativeKeywords = append(input.AdGroupNegativeKeywords, appleads.ParseNegativeKeywords(adGroupNegativeBroad, "BROAD")...)
	input.Creative = parseCreativeSelection(creative, creativeID, productPageID, creativeName, adName)
	input.Execute = (execute || yes) && !dryRun
	return input, nil
}

func buildProvider(ctx context.Context, cfg config.Config, verbose bool, logWriter io.Writer, willExecute bool) (appleads.Provider, appleads.RequestContext, error) {
	var provider appleads.Provider
	var baseURL string
	var contextHeader string

	switch strings.ToLower(cfg.Provider) {
	case "campaignv5", "":
		provider = campaignv5.New(cfg.OrgID)
		baseURL = cfg.CampaignV5Base
		contextHeader = "orgId=" + cfg.OrgID
	case "platform":
		provider = platform.New()
		baseURL = cfg.PlatformBase
		contextHeader = "adAccountId=" + cfg.AdAccountID
	default:
		return nil, nil, fmt.Errorf("unsupported provider %q", cfg.Provider)
	}

	if !willExecute {
		return provider, dryRunClient{}, nil
	}
	if err := cfg.ValidateAuth(); err != nil {
		return nil, nil, err
	}
	if err := cfg.ValidateProviderScope(); err != nil {
		return nil, nil, err
	}
	tokenSource, err := auth.NewTokenSource(cfg)
	if err != nil {
		return nil, nil, err
	}
	_ = ctx
	client := appleads.NewClient(baseURL, contextHeader, cfg.Timeout(), tokenSource)
	client.Verbose = verbose
	client.LogWriter = logWriter
	return provider, client, nil
}

func buildOpportunityProvider(ctx context.Context, cfg config.Config, verbose bool, logWriter io.Writer) (appleads.OpportunityProvider, appleads.RequestContext, error) {
	if config.NormalizeProvider(cfg.Provider) != "platform" {
		return nil, nil, fmt.Errorf("this command requires Apple Ads Platform API 1.0; add --provider platform")
	}
	provider, client, err := buildProvider(ctx, cfg, verbose, logWriter, true)
	if err != nil {
		return nil, nil, err
	}
	opportunityProvider, ok := provider.(appleads.OpportunityProvider)
	if !ok {
		return nil, nil, fmt.Errorf("provider %q does not support Platform opportunity queries", provider.Name())
	}
	return opportunityProvider, client, nil
}

type dryRunClient struct{}

func (dryRunClient) Do(method string, path string, body any) (appleads.RawResponse, error) {
	return nil, fmt.Errorf("dry-run client cannot execute %s %s", method, path)
}

func printValue(stdout io.Writer, format string, value any, stderr io.Writer) int {
	switch strings.ToLower(format) {
	case "", "json":
		if err := output.JSON(stdout, value); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unsupported output format %q; only json is implemented in this preview\n", format)
		return 2
	}
}

func parseCreativeSelection(mode string, creativeID string, productPageID string, creativeName string, adName string) appleads.CreativeSelection {
	selection := appleads.CreativeSelection{
		CreativeID:    strings.TrimSpace(creativeID),
		ProductPageID: strings.TrimSpace(productPageID),
		Name:          strings.TrimSpace(creativeName),
		AdName:        strings.TrimSpace(adName),
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		if selection.CreativeID != "" {
			selection.Kind = "creative-id"
		}
		if selection.ProductPageID != "" {
			selection.Kind = "product-page"
		}
		return selection
	}
	if before, after, ok := strings.Cut(mode, ":"); ok {
		mode = strings.TrimSpace(before)
		value := strings.TrimSpace(after)
		switch mode {
		case "product-page", "cpp":
			mode = "product-page"
			if selection.ProductPageID == "" {
				selection.ProductPageID = value
			}
		case "creative-id", "creative":
			mode = "creative-id"
			if selection.CreativeID == "" {
				selection.CreativeID = value
			}
		}
	}
	if mode == "cpp" {
		mode = "product-page"
	}
	selection.Kind = mode
	return selection
}

func defaultReportRange(input appleads.CampaignReportQuery) appleads.CampaignReportQuery {
	const layout = "2006-01-02"
	today := time.Now().Format(layout)
	if input.To == "" {
		input.To = today
	}
	if input.From == "" {
		toTime, err := time.Parse(layout, input.To)
		if err != nil {
			input.From = today
			return input
		}
		input.From = toTime.AddDate(0, 0, -6).Format(layout)
	}
	return input
}

func printHelp(w io.Writer) {
	output.Text(w,
		"Apple Ads CLI by Lily",
		"",
		"Usage:",
		"  lily login --token <token>",
		"  lily logout",
		"  lily auth status",
		"  lily ads doctor",
		"  lily ads platform readiness",
		"  lily ads plan recommend [flags]",
		"  lily ads plan create [flags]",
		"  lily ads reports campaigns [flags]",
		"  lily ads revenue summary [flags]",
		"  lily --provider platform ads suggestions keywords --app-id <adamId>",
		"  lily --provider platform ads suggestions target-cpa --app-id <adamId>",
		"  lily --provider platform ads recommendations <keywords|target-cpa|daily-budget> --app-id <adamId>",
		"  lily --provider platform ads insights <search-term-popularity|impression-share> [flags]",
		"  lily --provider platform ads change-history <query|detail> [flags]",
		"",
		"Global flags:",
		"  --provider campaignv5|platform   API provider (default: campaignv5)",
		"  -o, --output json                output format",
		"  -v, --verbose                    verbose API logging",
		"",
		"Auth model:",
		"  Apple Ads API credentials are configured locally. Private keys stay on this machine.",
		"  Lily login is optional and only enables Lily Ads Revenue Analytics revenue/ROAS enrichment.",
	)
}

func printAdsHelp(w io.Writer) {
	output.Text(w,
		"Usage:",
		"  lily ads doctor",
		"  lily ads platform readiness",
		"  lily ads plan recommend [flags]",
		"  lily ads plan create [flags]",
		"  lily ads reports campaigns [flags]",
		"  lily ads revenue summary [flags]",
		"  lily --provider platform ads suggestions keywords --app-id <adamId>",
		"  lily --provider platform ads suggestions target-cpa --app-id <adamId>",
		"  lily --provider platform ads recommendations <keywords|target-cpa|daily-budget> --app-id <adamId>",
		"  lily --provider platform ads insights <search-term-popularity|impression-share> [flags]",
		"  lily --provider platform ads change-history <query|detail> [flags]",
		"",
		"Apple Ads commands use local Apple Ads API credentials. Run `lily ads doctor` to check setup.",
		"Lily login is optional and only enables Lily Ads Revenue Analytics enrichment.",
	)
}

func printPlanHelp(w io.Writer) {
	output.Text(w,
		"Usage:",
		"  lily ads plan recommend --app-id <adamId> --country GB",
		"  lily ads plan create --name <name> --app-id <adamId> --country GB --daily-budget 300 --bid 2.00 --keywords \"photo editor,best photo editor\"",
		"",
		"Business-first workflow:",
		"  recommend creates a review-only draft with safe assumptions.",
		"  create prepares the same campaign package and remains dry-run unless --yes or --execute is set.",
		"  the package includes one campaign, one ad group, bulk keywords, optional negatives, and optional creative/ad attachment.",
		"",
		"Safety:",
		"  dry-run is the default. Add --yes or --execute to call Apple Ads.",
		"  missing budget, bid, or keywords are allowed only for review-only drafts; writes require explicit values.",
		"",
		"Flags:",
		"  --name <name>",
		"  --app-id <adamId>",
		"  --country <code> / --countries <codes>",
		"  --daily-budget <amount>",
		"  --currency <code>",
		"  --adgroup <name>",
		"  --bid <amount>",
		"  --cpa-goal <amount>",
		"  --keywords <kw1,kw2>",
		"  --exact-keywords <kw1,kw2>",
		"  --broad-keywords <kw1,kw2>",
		"  --negative-exact <kw1,kw2>",
		"  --negative-broad <kw1,kw2>",
		"  --campaign-negative-exact <kw1,kw2>",
		"  --campaign-negative-broad <kw1,kw2>",
		"  --adgroup-negative-exact <kw1,kw2>",
		"  --adgroup-negative-broad <kw1,kw2>",
		"  --creative none|default|product-page|product-page:<id>|creative-id:<id>",
		"  --product-page-id <id>",
		"  --creative-id <id>",
		"  --creative-name <name>",
		"  --ad-name <name>",
		"  --start-time <time>",
		"  --end-time <time>",
		"  --allow-partial",
		"  --correlation-id <id>",
		"  --yes",
	)
}
