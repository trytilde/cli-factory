package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"cli-factory/internal/discovery"
	"cli-factory/internal/envfile"
	"cli-factory/internal/invocationlog"
	"cli-factory/internal/openai"
	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
)

type App struct {
	Registry *provider.Registry
	Stdout   io.Writer
	Stderr   io.Writer
	Embedder discovery.Embedder
}

type globalFlags struct {
	Debug        bool
	LogDir       string
	NoLogCleanup bool
	OpenAIAPIKey string
}

func (a App) Run(ctx context.Context, args []string) int {
	if a.Stdout == nil {
		a.Stdout = os.Stdout
	}
	if a.Stderr == nil {
		a.Stderr = os.Stderr
	}
	flags, rest, err := parseGlobal(args)
	if err != nil {
		return a.finish(ctx, flags, args, nil, nil, errObj("validation_failed", err.Error(), false), 2)
	}
	wd, err := os.Getwd()
	if err != nil {
		return a.finish(ctx, flags, args, nil, nil, errObj("validation_failed", "get working directory: "+err.Error(), false), 2)
	}
	if err := envfile.LoadOptional(wd, ".env", ".env.secrets"); err != nil {
		return a.finish(ctx, flags, args, nil, nil, errObj("validation_failed", "load env files: "+err.Error(), false), 2)
	}
	if !flags.NoLogCleanup {
		dir := flags.LogDir
		if dir == "" {
			dir, _ = invocationlog.DefaultDir()
		}
		_ = invocationlog.Cleanup(dir, time.Now())
		stop := startCleanupWorker(dir)
		defer stop()
	}
	return a.dispatch(ctx, flags, args, rest)
}

func startCleanupWorker(dir string) func() {
	done := make(chan struct{})
	ticker := time.NewTicker(time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = invocationlog.Cleanup(dir, time.Now())
			case <-done:
				return
			}
		}
	}()
	return func() { close(done) }
}

func (a App) dispatch(ctx context.Context, flags globalFlags, original, args []string) int {
	if len(args) == 0 {
		return a.finish(ctx, flags, original, nil, nil, errObj("validation_failed", "command is required", false), 2)
	}
	switch args[0] {
	case "help", "--help", "-h":
		a.printHelp()
		return 0
	case "search":
		return a.runSearch(ctx, flags, original, args[1:])
	case "discover":
		return a.runDiscover(ctx, flags, original, args[1:])
	case "invoke":
		return a.runInvoke(ctx, flags, original, args[1:])
	default:
		if len(args) >= 2 {
			return a.runProviderTool(ctx, flags, original, args[0], args[1], args[2:])
		}
		return a.finish(ctx, flags, original, nil, nil, errObj("validation_failed", "unknown command", false), 2)
	}
}

func (a App) printHelp() {
	fmt.Fprintln(a.Stdout, "CLI Factory discovers and invokes curated SaaS/tool provider commands.")
	fmt.Fprintln(a.Stdout)
	fmt.Fprintln(a.Stdout, "Usage:")
	fmt.Fprintln(a.Stdout, "  factory search <query>")
	fmt.Fprintln(a.Stdout, "  factory discover short|long <provider> [tool]")
	fmt.Fprintln(a.Stdout, "  factory invoke <provider.tool> [--provider-params-json JSON] [--params-json JSON]")
	fmt.Fprintln(a.Stdout, "  factory <provider> <tool> [--param value] [--provider-param value]")
	fmt.Fprintln(a.Stdout, "  factory help")
	fmt.Fprintln(a.Stdout)
	fmt.Fprintln(a.Stdout, "Global flags:")
	fmt.Fprintln(a.Stdout, "  --debug            Print full result or error JSON")
	fmt.Fprintln(a.Stdout, "  --log-dir <path>   Write invocation logs to a custom directory")
	fmt.Fprintln(a.Stdout, "  --openai-api-key   API key for semantic search embeddings")
	fmt.Fprintln(a.Stdout)
	fmt.Fprintln(a.Stdout, "Commands:")
	fmt.Fprintln(a.Stdout, "  search             Find providers and tools by intent")
	fmt.Fprintln(a.Stdout, "  discover           Show provider/tool metadata and schemas")
	fmt.Fprintln(a.Stdout, "  invoke             Invoke a tool by dotted id")
	fmt.Fprintln(a.Stdout, "  <provider> <tool>  Invoke a tool with command-style flags")
	fmt.Fprintln(a.Stdout, "  help               Show this help")
	fmt.Fprintln(a.Stdout)
	fmt.Fprintln(a.Stdout, "Providers and tools:")
	for _, p := range a.Registry.Providers() {
		fmt.Fprintf(a.Stdout, "  %s - %s\n", p.ID(), p.ShortDescription())
		for _, tool := range p.Tools() {
			fmt.Fprintf(a.Stdout, "    %s - %s\n", tool.ID(), tool.ShortDescription())
		}
	}
}

func (a App) runSearch(ctx context.Context, flags globalFlags, original, args []string) int {
	if len(args) == 0 {
		return a.finish(ctx, flags, original, nil, nil, errObj("validation_failed", "search query is required", false), 2)
	}
	embedder := a.Embedder
	if embedder == nil {
		embedder = openai.Embedder{APIKey: flags.OpenAIAPIKey}
	}
	results, err := discovery.Search(ctx, a.Registry, embedder, strings.Join(args, " "))
	if err != nil {
		return a.finish(ctx, flags, original, nil, nil, errObj("provider_failed", err.Error(), true), 5)
	}
	return a.finish(ctx, flags, original, nil, results, nil, 0)
}

func (a App) runDiscover(ctx context.Context, flags globalFlags, original, args []string) int {
	if len(args) < 2 {
		return a.finish(ctx, flags, original, nil, nil, errObj("validation_failed", "usage: factory discover short|long <provider> [tool]", false), 2)
	}
	depth, providerID := args[0], args[1]
	if depth != "short" && depth != "long" {
		return a.finish(ctx, flags, original, nil, nil, errObj("validation_failed", "discover depth must be short or long", false), 2)
	}
	p, ok := a.Registry.Provider(providerID)
	if !ok {
		return a.finish(ctx, flags, original, nil, nil, errObj("validation_failed", "provider not found: "+providerID, false), 2)
	}
	if len(args) >= 3 {
		rt, ok := a.Registry.Tool(providerID, args[2])
		if !ok {
			return a.finish(ctx, flags, original, providerToolIDs(providerID, args[2]), nil, errObj("validation_failed", "tool not found: "+args[2], false), 2)
		}
		return a.finish(ctx, flags, original, providerToolIDs(providerID, args[2]), discoverTool(depth, rt.Provider, rt.Tool), nil, 0)
	}
	return a.finish(ctx, flags, original, []string{providerID}, discoverProvider(depth, p), nil, 0)
}

func (a App) runInvoke(ctx context.Context, flags globalFlags, original, args []string) int {
	if len(args) == 0 {
		return a.finish(ctx, flags, original, nil, nil, errObj("validation_failed", "tool id is required", false), 2)
	}
	toolID := args[0]
	parts := strings.SplitN(toolID, ".", 2)
	if len(parts) != 2 {
		return a.finish(ctx, flags, original, nil, nil, errObj("validation_failed", "tool id must be <provider>.<tool>", false), 2)
	}
	providerParamsJSON, paramsJSON := a.parseInvocationArgs(parts[0], args[1:])
	return a.invokeTool(ctx, flags, original, parts[0], parts[1], providerParamsJSON, paramsJSON)
}

func (a App) runProviderTool(ctx context.Context, flags globalFlags, original []string, providerID, toolID string, args []string) int {
	providerParamsJSON, paramsJSON := a.parseInvocationArgs(providerID, args)
	return a.invokeTool(ctx, flags, original, providerID, toolID, providerParamsJSON, paramsJSON)
}

func (a App) parseInvocationArgs(providerID string, args []string) (string, string) {
	params := map[string]any{}
	providerParams := map[string]any{}
	var paramsJSON, providerParamsJSON string
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") {
			continue
		}
		name := strings.TrimPrefix(args[i], "--")
		if name == "provider-params-json" {
			if i+1 < len(args) {
				i++
				providerParamsJSON = args[i]
			}
			continue
		}
		if name == "params-json" {
			if i+1 < len(args) {
				i++
				paramsJSON = args[i]
			}
			continue
		}
		value := "true"
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			i++
			value = args[i]
		}
		if isProviderParam(a.Registry, providerID, name) {
			providerParams[strings.ReplaceAll(name, "-", "_")] = value
		} else {
			params[strings.ReplaceAll(name, "-", "_")] = value
		}
	}
	if providerParamsJSON == "" {
		pp, _ := json.Marshal(providerParams)
		providerParamsJSON = string(pp)
	}
	if paramsJSON == "" {
		p, _ := json.Marshal(params)
		paramsJSON = string(p)
	}
	return providerParamsJSON, paramsJSON
}

func (a App) invokeTool(ctx context.Context, flags globalFlags, original []string, providerID, toolID, providerParamsJSON, paramsJSON string) int {
	rt, ok := a.Registry.Tool(providerID, toolID)
	if !ok {
		return a.finish(ctx, flags, original, providerToolIDs(providerID, toolID), nil, errObj("validation_failed", "tool not found", false), 2)
	}
	req := provider.InvokeRequest{ProviderParams: map[string]any{}, Params: map[string]any{}}
	if providerParamsJSON != "" {
		if err := json.Unmarshal([]byte(providerParamsJSON), &req.ProviderParams); err != nil {
			return a.finish(ctx, flags, original, providerToolIDs(providerID, toolID), nil, errObj("validation_failed", "invalid provider params json: "+err.Error(), false), 2)
		}
	}
	if paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &req.Params); err != nil {
			return a.finish(ctx, flags, original, providerToolIDs(providerID, toolID), nil, errObj("validation_failed", "invalid params json: "+err.Error(), false), 2)
		}
	}
	if err := validateRequiredInvocation(rt.Provider, rt.Tool, req); err != nil {
		return a.finish(ctx, flags, original, providerToolIDs(providerID, toolID), nil, errObj("validation_failed", err.Error(), false), 2)
	}
	rec, err := invocationlog.New(flags.LogDir, redactProviderSecrets(rt.Provider, original))
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return 1
	}
	rec.Provider, rec.Tool = providerID, toolID
	result, invokeErr := rt.Tool.Invoke(ctx, req, rec)
	if invokeErr != nil {
		perr := asProviderError(invokeErr)
		_ = rec.Finish("FAILURE", 4, nil, perr)
		a.printCompletion(flags, rec, nil, perr)
		return 4
	}
	_ = rec.Finish("SUCCESS", 0, result, nil)
	a.printCompletion(flags, rec, result, nil)
	return 0
}

func (a App) finish(_ context.Context, flags globalFlags, original []string, ids []string, result any, perr *provider.Error, exit int) int {
	rec, err := invocationlog.New(flags.LogDir, original)
	if err != nil {
		fmt.Fprintln(a.Stderr, err)
		return 1
	}
	if len(ids) > 0 {
		rec.Provider = ids[0]
	}
	if len(ids) > 1 {
		rec.Tool = ids[1]
	}
	status := "SUCCESS"
	if perr != nil {
		status = "FAILURE"
	}
	_ = rec.Finish(status, exit, result, perr)
	a.printCompletion(flags, rec, result, perr)
	return exit
}

func (a App) printCompletion(flags globalFlags, rec *invocationlog.Recorder, result any, perr *provider.Error) {
	if flags.Debug {
		if perr != nil {
			_ = json.NewEncoder(a.Stderr).Encode(map[string]any{"error": perr})
			return
		}
		_ = json.NewEncoder(a.Stdout).Encode(result)
		return
	}
	if perr != nil {
		fmt.Fprintln(a.Stdout, "FAILURE")
		fmt.Fprintf(a.Stdout, "full logs at %s\n", rec.Path)
		return
	}
	fmt.Fprintln(a.Stdout, "SUCCESS")
	fmt.Fprintf(a.Stdout, "full logs at %s\n", rec.Path)
}

func parseGlobal(args []string) (globalFlags, []string, error) {
	var flags globalFlags
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--debug":
			flags.Debug = true
		case "--no-log-cleanup":
			flags.NoLogCleanup = true
		case "--log-dir":
			i++
			if i >= len(args) {
				return flags, nil, fmt.Errorf("--log-dir requires a value")
			}
			flags.LogDir = args[i]
		case "--openai-api-key":
			i++
			if i >= len(args) {
				return flags, nil, fmt.Errorf("--openai-api-key requires a value")
			}
			flags.OpenAIAPIKey = args[i]
		default:
			rest = append(rest, args[i])
		}
	}
	return flags, rest, nil
}

func discoverProvider(depth string, p provider.Provider) map[string]any {
	tools := []map[string]any{}
	for _, t := range p.Tools() {
		item := map[string]any{"id": t.ID(), "name": t.Name(), "short_description": t.ShortDescription()}
		if depth == "long" {
			item["long_description"] = t.LongDescription()
		}
		tools = append(tools, item)
	}
	out := map[string]any{
		"provider":          p.ID(),
		"name":              p.Name(),
		"short_description": p.ShortDescription(),
		"categories":        p.Categories(),
		"tools":             tools,
		"provider_params":   paramNames(p.Parameters()),
	}
	if depth == "long" {
		out["long_description"] = p.LongDescription()
		out["aliases"] = p.Aliases()
		out["provider_parameters"] = p.Parameters()
	}
	return out
}

func discoverTool(depth string, p provider.Provider, t provider.Tool) map[string]any {
	inputNames := schema.PropertyNames(t.InputSchema())
	outputNames := schema.PropertyNames(t.OutputSchema())
	sort.Strings(inputNames)
	sort.Strings(outputNames)
	out := map[string]any{
		"provider":          p.ID(),
		"tool":              t.ID(),
		"name":              t.Name(),
		"short_description": t.ShortDescription(),
		"categories":        t.Categories(),
		"provider_params":   strings.Join(paramNames(p.Parameters()), ", "),
		"input_params":      strings.Join(inputNames, ", "),
		"outputs":           strings.Join(outputNames, ", "),
	}
	if depth == "long" {
		out["long_description"] = t.LongDescription()
		out["aliases"] = t.Aliases()
		out["command_path"] = []string{p.ID(), t.ID()}
		out["input_schema"] = t.InputSchema()
		out["output_schema"] = t.OutputSchema()
	}
	return out
}

func validateRequiredInvocation(p provider.Provider, t provider.Tool, req provider.InvokeRequest) error {
	var missing []string
	for _, param := range p.Parameters() {
		if !param.Required {
			continue
		}
		if isMissing(req.ProviderParams[param.Name]) {
			missing = append(missing, param.Name)
		}
	}
	missing = append(missing, missingRequiredSchemaParams(t.InputSchema(), req.Params)...)
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("missing required parameters: %s", strings.Join(missing, ", "))
}

func missingRequiredSchemaParams(s schema.JSONSchema, params map[string]any) []string {
	required, ok := s["required"].([]any)
	if !ok {
		return nil
	}
	var missing []string
	for _, raw := range required {
		name, ok := raw.(string)
		if !ok {
			continue
		}
		if isMissing(params[name]) {
			missing = append(missing, name)
		}
	}
	return missing
}

func isMissing(value any) bool {
	if value == nil {
		return true
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

func isProviderParam(r *provider.Registry, providerID, flag string) bool {
	p, ok := r.Provider(providerID)
	if !ok {
		return false
	}
	name := strings.ReplaceAll(flag, "-", "_")
	for _, param := range p.Parameters() {
		if param.Name == name {
			return true
		}
	}
	return false
}

func paramNames(params []provider.Parameter) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Name)
	}
	sort.Strings(out)
	return out
}

func errObj(code, msg string, retryable bool) *provider.Error {
	return &provider.Error{Code: code, Message: msg, Retryable: retryable}
}

func asProviderError(err error) *provider.Error {
	if perr, ok := err.(*provider.Error); ok {
		return perr
	}
	return errObj("provider_failed", err.Error(), false)
}

func providerToolIDs(providerID, toolID string) []string {
	return []string{providerID, toolID}
}

func redactProviderSecrets(p provider.Provider, args []string) []string {
	if p == nil {
		return append([]string(nil), args...)
	}
	secrets := map[string]bool{}
	for _, param := range p.Parameters() {
		if param.Secret {
			secrets[param.Name] = true
			secrets[strings.ReplaceAll(param.Name, "_", "-")] = true
		}
	}
	if len(secrets) == 0 {
		return append([]string(nil), args...)
	}
	out := append([]string(nil), args...)
	for i := 0; i < len(out); i++ {
		name := strings.TrimPrefix(out[i], "--")
		switch name {
		case "provider-params-json":
			if i+1 < len(out) {
				out[i+1] = redactProviderParamsJSON(out[i+1], secrets)
				i++
			}
		default:
			if strings.HasPrefix(out[i], "--") && secrets[name] && i+1 < len(out) {
				out[i+1] = "[REDACTED]"
				i++
			}
		}
	}
	return out
}

func redactProviderParamsJSON(value string, secrets map[string]bool) string {
	params := map[string]any{}
	if err := json.Unmarshal([]byte(value), &params); err != nil {
		return "[REDACTED_PROVIDER_PARAMS]"
	}
	for name := range params {
		if secrets[name] || secrets[strings.ReplaceAll(name, "_", "-")] {
			params[name] = "[REDACTED]"
		}
	}
	data, err := json.Marshal(params)
	if err != nil {
		return "[REDACTED_PROVIDER_PARAMS]"
	}
	return string(data)
}
