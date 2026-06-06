// Package submit handles the submit flow for gou-demo.
package submit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"goc/commands"
	"goc/compactservice"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	ccbhydrate "goc/gou/ccbhydrate"
	config "goc/gou/app/config"
	"goc/gou/conversation"
	"goc/gou/pui"
	"goc/growthbook"
	"goc/hookexec"
	"goc/messagesapi"
	"goc/modelenv"
	"goc/tscontext"
	"goc/querycontext"
	"goc/services/autodream"
	"goc/services/extractmemories"
	"goc/services/sessionmemory"
	"goc/sessiontranscript"
	"goc/tools"
	"goc/tools/localtools"
	"goc/tools/skilltools"
	"goc/tools/toolexecution"
	"goc/tools/toolresultpersist"
	"goc/tools/toolsearchwire"
	"goc/types"
)

// Deps defines model dependencies for the submit flow.
type Deps interface {
	Model() tea.Model
	ConversationStore() *conversation.Store
	ConversationTSBridge() *tscontext.Snapshot
	ConversationReadFileState() *localtools.ReadFileState
	GroupedAgentLookups() interface{}
	ResolvedToolIDs() map[string]struct{}
	MCPCommandsJSONPath() string
	MCPToolsJSONPath() string
	ToolResultState() *toolresultpersist.ContentReplacementState
	SessionMem() *sessionmemory.State
	CCBSend() func(interface{})
	CCBInline() bool
	ModalAskAutoFirst() bool
	SkillListingSent() map[string]struct{}
	SetSkillListingSent(v map[string]struct{})
	PRSetValue(v string)
	ScrollSetSticky(v bool)
	ScrollSetTop(v int)
	QuerySetCancel(v context.CancelFunc)
	QuerySetBusy(v bool)
	QuerySetBusyStartedAt(v time.Time)
	QuerySetSpinnerVerb(v string)
	QuerySetSpinnerFrame(v int)
	RebuildHeightCache()
	MaybeRecordTranscript()
	SyncSlashListAfterPrompt()
	ApplySlashResultPanelFromSubmit(line string, r *processuserinput.ProcessUserInputBaseResult, out pui.ApplyProcessUserInputBaseResultOutcome)
	InstallAskResolver(te *toolexecution.ExecutionDeps, askAutoFirst bool)
	MessageBodyColsForLayout() int
	MessageScrollContentHeight() int
	IntegrateMessageRenderer()
	FillMessageHeightCache(cols int, hl string)
	BeginQuerySpinner()
	EndQuerySpinner()
	LoadSlashCommandsOnce()
	LastGuidance() string
	SetLastGuidance(v string)
	LastUserCtx() map[string]string
	SetLastUserCtx(v map[string]string)
	LastSystemCtx() map[string]string
	SetLastSystemCtx(v map[string]string)
	PermissionMode() types.PermissionMode
	ScreenMode() int
	LayoutCols() int
	LayoutHeight() int
	ConversationTranscript() *sessiontranscript.Store
}

// Submitter handles the submit flow.
type Submitter struct {
	deps Deps
}

// New creates a new Submitter.
func New(deps Deps) *Submitter {
	return &Submitter{deps: deps}
}

// Submit runs the submit flow, mirroring gouSubmitFromPromptText.
func (s *Submitter) Submit(fullPrompt string) (tea.Model, tea.Cmd) {
	d := s.deps
	line := strings.TrimSpace(fullPrompt)
	var cmd tea.Cmd

	cwd, _ := os.Getwd()
	toolProjectRoot := resolveToolProjectRoot(cwd)
	mergedLang, mergedOutName, mergedOutPrompt := mergedSystemLocale()

	preExp := fullPrompt
	pMode := d.PermissionMode()
	demoCfg := pui.DemoConfig{
		SessionID:           d.ConversationStore().ConversationID,
		Language:            mergedLang,
		MCPCommandsJSONPath: d.MCPCommandsJSONPath(),
		MCPToolsJSONPath:    d.MCPToolsJSONPath(),
		PreExpansionInput:   &preExp,
		PermissionMode:      &pMode,
	}
	if tsb := d.ConversationTSBridge(); tsb != nil {
		demoCfg.TSContextBridge = tsb
	}

	params, err := pui.BuildDemoParams(line, d.ConversationStore(), demoCfg)
	if err != nil {
		d.ConversationStore().AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: build params: %v", err)))
		d.RebuildHeightCache()
		d.ScrollSetSticky(true)
		d.ScrollSetTop(1 << 30)
		return d.Model(), cmd
	}
	if params.RuntimeContext != nil {
		_ = params.RuntimeContext
	}
	guidanceP := d.LastGuidance()
	userCtxP := d.LastUserCtx()
	sysCtxP := d.LastSystemCtx()
	params.ProcessSlashCommand = pui.NewSlashResolveProcessSlashCommand(pui.SlashResolveHandlerOptions{
		SessionID:        d.ConversationStore().ConversationID,
		Store:            d.ConversationStore(),
		ReadFileState:    d.ConversationReadFileState(),
		Cwd:              cwd,
		SessionMemState:  d.SessionMem(),
		GuidancePtr:      &guidanceP,
		UserContextPtr:   &userCtxP,
		SystemContextPtr: &sysCtxP,
	})

	r, err := processuserinput.ProcessUserInput(context.Background(), params)
	if err != nil {
		d.ConversationStore().AppendMessage(pui.SystemNotice(fmt.Sprintf("processUserInput: %v", err)))
		d.RebuildHeightCache()
		d.ScrollSetSticky(true)
		d.ScrollSetTop(1 << 30)
		return d.Model(), cmd
	}

	rStore := r
	if r != nil && strings.HasPrefix(line, "/") && r.Execution == nil && !r.ShouldQuery &&
		extractSlashLocalPanelText(r) != "" {
		rStore = slashResultForStoreOmittingPanelDupes(r)
	}

	out := pui.ApplyBaseResult(d.ConversationStore(), rStore, nil)
	if out.NextInput != "" {
		d.PRSetValue(out.NextInput)
		d.SyncSlashListAfterPrompt()
	}
	d.ApplySlashResultPanelFromSubmit(line, r, out)
	d.RebuildHeightCache()
	d.ScrollSetSticky(true)
	d.ScrollSetTop(1 << 30)
	d.MaybeRecordTranscript()

	if out.EffectiveShouldQuery && !out.HadExecutionRequest {
		usedCCB := false
		var normToolsJSON json.RawMessage
		if params.RuntimeContext != nil {
			normToolsJSON = params.RuntimeContext.ToolUseContext.Options.Tools
		}
		var normToolDefs []struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(normToolsJSON, &normToolDefs)
		toolSpecs := make([]messagesapi.ToolSpec, 0, len(normToolDefs))
		for _, t := range normToolDefs {
			toolSpecs = append(toolSpecs, messagesapi.ToolSpec{Name: t.Name})
		}
		normOpts := messagesapi.OptionsFromEnv()
		if envTruthy("GOU_DEMO_NON_INTERACTIVE") {
			normOpts.NonInteractive = true
		}
		tryMsgs := func() (json.RawMessage, error) {
			return ccbhydrate.MessagesJSONNormalized(d.ConversationStore().Messages, toolSpecs, normOpts)
		}

		if d.CCBInline() && d.CCBSend() != nil {
			baseMsgs, err := tryMsgs()
			if err != nil {
				d.ConversationStore().AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: ccb messages JSON: %v", err)))
				d.RebuildHeightCache()
			} else if len(strings.TrimSpace(string(baseMsgs))) < 3 {
				d.ConversationStore().AppendMessage(pui.SystemNotice("gou-demo: empty chat transcript (cannot call model)"))
				d.RebuildHeightCache()
			} else {
				var toolsJSON json.RawMessage
				if params.RuntimeContext != nil {
					toolsJSON = params.RuntimeContext.ToolUseContext.Options.Tools
				}
				var toolDefs []struct {
					Name string `json:"name"`
				}
				_ = json.Unmarshal(toolsJSON, &toolDefs)
				names := make([]string, 0, len(toolDefs))
				for _, t := range toolDefs {
					names = append(names, t.Name)
				}
				hasSkillTool := false
				skillNm := skilltools.SkillToolName()
				for _, t := range toolDefs {
					if t.Name == skillNm {
						hasSkillTool = true
						break
					}
				}
				skillListing := params.SkillListingCommands
				if len(skillListing) == 0 {
					skillListing = commands.SkillToolCommands(params.Commands)
				}
				discoverNm := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
				mainLoopModel := modelenv.EffectiveMainLoopModel()

				gouOpts := commands.GouDemoSystemOpts{
					EnabledToolNames:       commands.EnabledToolNames(names),
					SkillToolCommands:      skillListing,
					ModelID:                mainLoopModel,
					Cwd:                    cwd,
					Language:               mergedLang,
					DiscoverSkillsToolName: discoverNm,
					NonInteractiveSession:  envTruthy("GOU_DEMO_NON_INTERACTIVE"),
					OutputStyleName:        mergedOutName,
					OutputStylePrompt:      mergedOutPrompt,
				}
				commands.ApplyGouDemoRuntimeEnv(&gouOpts)

				var customSys, appendSys string
				if params.RuntimeContext != nil {
					if p := params.RuntimeContext.ToolUseContext.Options.CustomSystemPrompt; p != nil {
						customSys = strings.TrimSpace(*p)
					}
					if p := params.RuntimeContext.ToolUseContext.Options.AppendSystemPrompt; p != nil {
						appendSys = strings.TrimSpace(*p)
					}
				}
				extraRoots := querycontext.ExtraClaudeMdRootsForFetch(params.RuntimeContext)
				fetchOpts := querycontext.FetchOpts{
					CustomSystemPrompt:  customSys,
					Gou:                 gouOpts,
					ExtraClaudeMdRoots:  extraRoots,
					SessionStartSource:  "startup",
					HooksSessionID:      d.ConversationStore().ConversationID,
					HooksTranscriptPath: "",
				}
				if tsb := d.ConversationTSBridge(); tsb != nil {
					fetchOpts.TSSnapshot = tsb
				}

				partsRes, errParts := querycontext.FetchSystemPromptParts(context.Background(), fetchOpts)
				var guidance string
				if errParts != nil {
					d.ConversationStore().AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: system context: %v (using base prompt only)", errParts)))
					d.RebuildHeightCache()
					guidance = commands.BuildGouDemoSystemPrompt(gouOpts)
				} else {
					var base []string
					if customSys != "" {
						base = []string{customSys}
					} else {
						base = slices.Clone(partsRes.DefaultSystemPrompt)
					}
					if appendSys != "" {
						base = append(base, appendSys)
					}
					guidance = strings.Join(base, "\n\n")
				}
				d.SetLastGuidance(guidance)

				listing := ""
				var listingMeta *ccbhydrate.SkillListingMeta
				if !envTruthy("GOU_DEMO_SKIP_SKILL_LISTING") {
					listingSent := d.SkillListingSent()
					if envTruthy("GOU_DEMO_SKILL_LISTING_EVERY_TURN") {
						d.SetSkillListingSent(make(map[string]struct{}))
						listingSent = make(map[string]struct{})
					}
					if s, n, initial, ok := commands.AppendSkillListingForAPI(skillListing, hasSkillTool, listingSent, nil); ok {
						listing = s
						listingMeta = &ccbhydrate.SkillListingMeta{SkillCount: n, IsInitial: initial}
					}
				}

				msgsJSON, errL := ccbhydrate.MessagesJSONWithLeadingMeta(d.ConversationStore().Messages, "", listing, listingMeta, toolSpecs, normOpts)
				if errL != nil {
					d.ConversationStore().AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: skill listing hydrate: %v", errL)))
					d.RebuildHeightCache()
				} else {
					msgsBefore := len(msgsJSON)
					if prep := toolsearchwire.PrepareMessagesForWire(msgsJSON, toolsJSON, mainLoopModel, false, false, d.ConversationStore().Messages); len(prep) > 0 {
						msgsJSON = prep
					}
					if len(msgsJSON) > msgsBefore {
						toolsearchwire.PersistDeferredAnnouncement(d.ConversationStore(), toolsJSON)
					}
					d.ConversationStore().ClearStreaming()
					d.ConversationStore().ClearStreamingToolUses()
					if strings.TrimSpace(listing) != "" {
						if att, ok := ccbhydrate.SkillListingStoreMessage(listing, listingMeta); ok {
							d.ConversationStore().AppendMessage(att)
							d.RebuildHeightCache()
						}
					}

					cwdAbs, errAbs := filepath.Abs(cwd)
					if errAbs != nil {
						cwdAbs = cwd
					}
					runner := skilltools.ParityToolRunner{
						DemoToolRunner: skilltools.DemoToolRunner{
							Commands:  params.Commands,
							SessionID: d.ConversationStore().ConversationID,
						},
						WorkDir:          cwdAbs,
						ProjectRoot:      toolProjectRoot,
						ReadFileState:    d.ConversationReadFileState(),
						LocalBashDefault: true,
						AskAutoFirst:     !envTruthy("GOU_DEMO_NO_ASK_AUTO_FIRST"),
						MainLoopModel:    mainLoopModel,
						Messages:         d.ConversationStore().Messages,
						MessagesFunc:     func() []types.Message { return d.ConversationStore().Messages },
						SystemPrompt:     []string{guidance},
						ProgressCallback: func(msg *types.Message) {
							if msg == nil {
								return
							}
							if send := d.CCBSend(); send != nil {
								send(msg)
							}
						},
						EditDeps: &localtools.EditDeps{
							OnNotebookEdit: func(absPath, oldString, newString string, replaceAll bool, roots []string, state *localtools.ReadFileState, userModified bool) (string, bool, error) {
								return tools.NotebookEditFromEdit(absPath, oldString, newString, replaceAll, roots)
							},
						},
					}
					if params.RuntimeContext != nil && params.RuntimeContext.ToolPermissionContext != nil {
						runner.ToolPermission = params.RuntimeContext.ToolPermissionContext
					}

					if envTruthy("GOU_QUERY_STREAMING_PARITY") || envTruthy("GOU_DEMO_STREAMING_TOOL_EXECUTION") {
						userCtx := d.LastUserCtx()
						systemCtx := d.LastSystemCtx()
						tcx := types.ToolUseContext{}
						if params.RuntimeContext != nil {
							tcx = params.RuntimeContext.ToolUseContext
						}
						tcx.Options.MainLoopModel = mainLoopModel
						if d.ToolResultState() != nil {
							tcx.ContentReplacementState = d.ToolResultState().ToJSON()
						}
						var trySMCompact compactservice.TrySessionMemoryCompactFn
						if d.SessionMem() != nil {
							sessionID := d.ConversationStore().ConversationID
							trySMCompact = func(ctx context.Context, messages []types.Message, agentID string, autoCompactThreshold *int) (*compactservice.CompactionResult, error) {
								return sessionmemory.TrySessionMemoryCompaction(
									ctx,
									d.SessionMem(),
									sessionID,
									cwd,
									messages,
									"",
									autoCompactThreshold,
									func(ctx context.Context, trigger string, model string) ([]types.Message, error) {
										runner := hookexec.SessionStartHookRunner(toolProjectRoot, cwd, sessionID, "")
										res, err := runner(ctx, compactservice.SessionStartHookTrigger(trigger),
											compactservice.SessionStartHookInput{Model: model})
										if err != nil {
											return nil, err
										}
										msgs := make([]types.Message, len(res))
										for i, r := range res {
											msgs[i] = r
										}
										return msgs, nil
									},
									agentID,
									mainLoopModel,
									nil,
								)
							}
						}
						qdeps := query.ProductionDeps(trySMCompact, func(phase string) {
							if send := d.CCBSend(); send != nil {
								send(phase)
							}
						})
						te := toolexecution.ExecutionDeps{
							InvokeTool:              runner.Run,
							MainLoopModel:           mainLoopModel,
							ReadToolRoots:           runner.ToolReadMappingRoots(),
							ReadToolMemCWD:          runner.ToolReadMappingMemCWD(),
							MultiMessageToolHandler: skilltools.NewSkillMultiMessageHandler(params.Commands, d.ConversationStore().ConversationID, nil),
							QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
								if toolName == "AskUserQuestion" {
									return toolexecution.AskDecision("Answer questions?"), nil
								}
								return toolexecution.AllowDecision(), nil
							},
						}
						if envTruthy("GOU_TOOLEXEC_BASH_SANDBOX_1B") {
							te.SandboxingEnabled = true
							te.AutoAllowBashWholeToolAskWhenSandboxed = true
						}
						if !envFalsy("GOU_DEMO_TOOL_RESULT_PERSIST") {
							te.ToolResultPersistConfig = &toolexecution.ToolResultPersistConfig{
								SessionInfo: toolresultpersist.SessionInfo{
									SessionID: d.ConversationStore().ConversationID,
									Cwd:       cwd,
								},
								ProcessOptions:          toolresultpersist.DefaultProcessOptions(),
								ContentReplacementState: d.ToolResultState(),
							}
						}
						d.InstallAskResolver(&te, d.ModalAskAutoFirst())
						qdeps.ToolexecutionDeps = te

						if d.ToolResultState() != nil {
							statePtr := d.ToolResultState()
							sessionInfo := te.ToolResultPersistConfig.SessionInfo
							qdeps.ApplyToolResultBudget = func(ctx context.Context, in *query.ToolResultBudgetInput) ([]types.Message, error) {
								return toolresultpersist.ApplyToolResultBudget(
									in.Messages,
									statePtr,
									sessionInfo,
									0, nil,
								), nil
							}
						}
						msgsForQ := slices.Clone(d.ConversationStore().Messages)
						if partsRes.SessionStartHookMessages != nil {
							msgsForQ = append(slices.Clone(partsRes.SessionStartHookMessages), msgsForQ...)
						}
						if send := d.CCBSend(); send != nil {
							qdeps.OnStreamingToolUses = func(ctx context.Context, uses []query.StreamingToolUseLive) error {
								send(uses)
								return nil
							}
						}
						tr := d.ConversationTranscript()
						if tr != nil && tr == nil {
							_ = tr
						}
						conversationTr := d.ConversationTranscript()
						if conversationTr != nil {
							turnPrefix := slices.Clone(msgsForQ)
							qdeps.OnQueryYield = func(ctx context.Context, y query.QueryYield) error {
								if y.Message == nil {
									return nil
								}
								turnPrefix = append(turnPrefix, *y.Message)
								_, err := conversationTr.RecordTranscript(ctx, turnPrefix, sessiontranscript.RecordOpts{AllMessages: turnPrefix})
								return err
							}
						}
						qdeps.OnQueryComplete = func(ctx context.Context, qcp query.QueryCompleteParams) {
							extractmemories.Execute(ctx, &extractmemories.State{}, extractmemories.ExtractionParams{
								Messages:       qcp.Messages,
								ToolUseContext: qcp.ToolUseContext,
								SystemPrompt:   qcp.SystemPrompt,
								UserContext:    qcp.UserContext,
								SystemContext:  qcp.SystemContext,
								Cwd:            qcp.Cwd,
								QuerySource:    qcp.QuerySource,
								NewUUID:        query.RandomUUID,
								SkipIndex:      growthbook.IsTenguMothCopse(),
								AppendSystemMessage: func(msg types.Message) {
									if send := d.CCBSend(); send != nil {
										send(msg)
									}
								},
							})
							_, dreamErr := autodream.Execute(context.Background(), &autodream.State{},
								qcp.ToolUseContext, qcp.SystemPrompt,
								qcp.UserContext, qcp.SystemContext,
								qcp.QuerySource, query.RandomUUID,
								commands.ClaudeConfigHome(), qcp.Cwd,
								"",
								d.ConversationStore().ConversationID,
							)
							if dreamErr != nil {
								_ = dreamErr
							}
						}
						qp := query.QueryParams{
							Messages:        msgsForQ,
							SystemPrompt:    query.AsSystemPrompt([]string{guidance}),
							UserContext:     userCtx,
							SystemContext:   systemCtx,
							ToolUseContext:  tcx,
							QuerySource:     params.QuerySource,
							StreamingParity: true,
							Deps:            &qdeps,
						}
						if params.RuntimeContext != nil && params.RuntimeContext.ToolPermissionContext != nil {
							pc := *params.RuntimeContext.ToolPermissionContext
							types.NormalizeToolPermissionContextData(&pc)
							qp.ToolPermissionContext = &pc
						}
						processuserinput.ApplyQueryHostEnvGates(&qp)
						processuserinput.WireToolexecutionFromProcessUserInput(&qp, params)

						d.ConversationStore().ClearStreamingToolUses()
						ctx, cancel := context.WithCancel(context.Background())
						d.QuerySetCancel(cancel)
						d.QuerySetBusy(true)
						d.QuerySetSpinnerVerb("Working")
						d.QuerySetBusyStartedAt(time.Now())
						d.QuerySetSpinnerFrame(0)

						// Run the query turn via the same config function as main.go.
						// The model's CCBSend handles the messages.
						programSend := func(msg tea.Msg) {
							if send := d.CCBSend(); send != nil {
								send(msg)
							}
						}
						runQueryStreamingParityTurn(ctx, programSend, qp)
						usedCCB = true
					} else {
						d.ConversationStore().AppendMessage(pui.SystemNotice(
							"gou-demo: ccb-engine/localturn was removed.",
						))
						d.RebuildHeightCache()
					}
				}
			}
			if usedCCB {
				if cmd != nil {
					return d.Model(), tea.Batch(cmd, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return nil }))
				}
				return d.Model(), tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return nil })
			}
			if !d.CCBInline() {
				d.ConversationStore().AppendMessage(pui.SystemNotice(
					"gou-demo: real HTTP / streaming parity is disabled (GOU_DEMO_CCB_INLINE=0).",
				))
				d.RebuildHeightCache()
				d.ScrollSetSticky(true)
				d.ScrollSetTop(1 << 30)
			}
			if cmd != nil {
				return d.Model(), cmd
			}
			return d.Model(), nil
		}
	}
	if cmd != nil {
		return d.Model(), cmd
	}
	return d.Model(), nil
}

func extractSlashLocalPanelText(r *processuserinput.ProcessUserInputBaseResult) string {
	if r == nil || len(r.Messages) == 0 {
		return ""
	}
	const inf = "informational"
	var parts []string
	for i := range r.Messages {
		msg := r.Messages[i]
		if msg.Type != types.MessageTypeSystem {
			continue
		}
		if msg.Subtype == nil || *msg.Subtype != inf {
			continue
		}
		var s string
		if json.Unmarshal(msg.Content, &s) != nil || strings.TrimSpace(s) == "" {
			continue
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n\n")
}

func slashResultForStoreOmittingPanelDupes(r *processuserinput.ProcessUserInputBaseResult) *processuserinput.ProcessUserInputBaseResult {
	if r == nil || len(r.Messages) == 0 {
		return r
	}
	const inf = "informational"
	var kept []types.Message
	for i := range r.Messages {
		msg := r.Messages[i]
		if msg.Type == types.MessageTypeSystem && msg.Subtype != nil && *msg.Subtype == inf {
			var s string
			if json.Unmarshal(msg.Content, &s) == nil && strings.TrimSpace(s) != "" {
				continue
			}
		}
		kept = append(kept, msg)
	}
	if len(kept) == len(r.Messages) {
		return r
	}
	out := *r
	out.Messages = kept
	return &out
}

func resolveToolProjectRoot(cwd string) string {
	return cwd
}

func mergedSystemLocale() (string, string, string) {
	lang := os.Getenv("CLAUDE_CODE_LANGUAGE")
	outName := os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_NAME")
	outPrompt := os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_PROMPT")
	return lang, outName, outPrompt
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" || v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	return true
}

func envFalsy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return true
	}
	return false
}

// runQueryStreamingParityTurn delegates to config.RunQueryStreamingParityTurn.
func runQueryStreamingParityTurn(ctx context.Context, programSend func(tea.Msg), qp query.QueryParams) {
	config.RunQueryStreamingParityTurn(ctx, programSend, qp)
}
