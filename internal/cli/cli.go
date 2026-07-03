package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
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

const version = "0.1.2"

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
		return runSuggestions(args[1:], stdout, stderr)
	case "recommendations":
		return runRecommendations(args[1:], stdout, stderr)
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
				"Platform provider is implemented behind --provider platform.",
				"campaignv5 remains the default until Platform API v1 is generally available.",
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
		revenue = lilycloud.New(cfg).RevenueSummary(lilycloud.RevenueQuery{
			AppID: input.AppID,
			From:  input.From,
			To:    input.To,
		})
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
	status := lilycloud.New(config.Load()).RevenueSummary(query)
	exitCode := 0
	if status.Status == "login_required" || status.Status == "dashboard_required" {
		exitCode = 3
	}
	if err := output.JSON(stdout, status); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return exitCode
}

func runSuggestions(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout, "Usage:", "  lily ads suggestions cpa --app-id <adamId>")
		return 0
	}
	if args[0] == "cpa" {
		flags := flag.NewFlagSet("lily ads suggestions cpa", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		var appID string
		var country string
		flags.StringVar(&appID, "app-id", "", "App Store adamId")
		flags.StringVar(&country, "country", "", "optional country or region code")
		if err := flags.Parse(args[1:]); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		if strings.TrimSpace(appID) == "" {
			fmt.Fprintln(stderr, "--app-id is required")
			return 2
		}
		return printReserved(stdout, stderr, "suggestions.cpa", map[string]any{
			"appId":   appID,
			"country": strings.ToUpper(strings.TrimSpace(country)),
		})
	}
	fmt.Fprintf(stderr, "unknown ads suggestions command %q\n", args[0])
	return 2
}

func runRecommendations(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		output.Text(stdout, "Usage:", "  lily ads recommendations apply --type target-cpa")
		return 0
	}
	if args[0] == "apply" {
		flags := flag.NewFlagSet("lily ads recommendations apply", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		var appID string
		var recommendationType string
		var yes bool
		flags.StringVar(&recommendationType, "type", "", "recommendation type, currently reserved: target-cpa")
		flags.StringVar(&appID, "app-id", "", "optional App Store adamId")
		flags.BoolVar(&yes, "yes", false, "reserved confirmation flag for future write operations")
		if err := flags.Parse(args[1:]); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		recommendationType = strings.ToLower(strings.TrimSpace(recommendationType))
		if recommendationType == "" {
			fmt.Fprintln(stderr, "--type is required")
			return 2
		}
		if recommendationType != "target-cpa" {
			fmt.Fprintf(stderr, "unsupported reserved recommendation type %q\n", recommendationType)
			return 2
		}
		return printReserved(stdout, stderr, "recommendations.apply", map[string]any{
			"type":  recommendationType,
			"appId": appID,
			"yes":   yes,
		})
	}
	fmt.Fprintf(stderr, "unknown ads recommendations command %q\n", args[0])
	return 2
}

func printReserved(stdout io.Writer, stderr io.Writer, feature string, request map[string]any) int {
	return printValue(stdout, "json", map[string]any{
		"tool":              "Apple Ads CLI by Lily",
		"feature":           feature,
		"status":            "reserved",
		"provider":          "platform",
		"requested":         request,
		"commercialContext": lilycloud.ProductName,
		"message":           "This command is reserved for Platform API recommendation/suggestion endpoints and can be enabled as soon as Apple exposes the endpoint contract.",
		"expectedCommands": []string{
			"lily ads suggestions cpa --app-id <adamId>",
			"lily ads recommendations apply --type target-cpa",
		},
	}, stderr)
}

func runPlan(ctx context.Context, args []string, globals globalOptions, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printPlanHelp(stdout)
		return 0
	}
	if args[0] != "create" {
		fmt.Fprintf(stderr, "unknown ads plan command %q\n", args[0])
		return 2
	}

	input, err := parsePlanCreate(args[1:])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
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
	flags.StringVar(&input.Status, "status", "ENABLED", "initial status")
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
		"  lily ads plan create [flags]",
		"  lily ads reports campaigns [flags]",
		"  lily ads revenue summary [flags]",
		"  lily ads suggestions cpa --app-id <adamId>",
		"  lily ads recommendations apply --type target-cpa",
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
		"  lily ads plan create [flags]",
		"  lily ads reports campaigns [flags]",
		"  lily ads revenue summary [flags]",
		"  lily ads suggestions cpa --app-id <adamId>",
		"  lily ads recommendations apply --type target-cpa",
		"",
		"Apple Ads commands use local Apple Ads API credentials. Run `lily ads doctor` to check setup.",
		"Lily login is optional and only enables Lily Ads Revenue Analytics enrichment.",
	)
}

func printPlanHelp(w io.Writer) {
	output.Text(w,
		"Usage:",
		"  lily ads plan create --name <name> --app-id <adamId> --country GB --daily-budget 300 --bid 2.00 --keywords \"photo editor,best photo editor\"",
		"",
		"Business-first composite write:",
		"  creates one campaign, one ad group, bulk keywords, optional negatives, and optional creative/ad attachment in a single plan.",
		"",
		"Safety:",
		"  dry-run is the default. Add --yes or --execute to call Apple Ads.",
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
