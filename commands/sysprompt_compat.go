// Package commands — sysprompt re-exports for backward compatibility.
// The implementation lives in commands/sysprompt/.
package commands

import "goc/commands/sysprompt"

// --- Constants ---

const (
	ClaudeCodeDocsMapURL       = sysprompt.ClaudeCodeDocsMapURL
	SystemPromptDynamicBoundary = sysprompt.SystemPromptDynamicBoundary
	FrontierModelName          = sysprompt.FrontierModelName
	DefaultAgentPrompt         = sysprompt.DefaultAgentPrompt
	TickTag                    = sysprompt.TickTag
	SleepToolName              = sysprompt.SleepToolName
	SummarizeToolResultsSection = sysprompt.SummarizeToolResultsSection
	CyberRiskInstruction       = sysprompt.CyberRiskInstruction
	BriefToolName              = sysprompt.BriefToolName
)

var (
	Claude45Or46ModelIDs  = sysprompt.Claude45Or46ModelIDs
	HarnessModelDisplayIDs = sysprompt.HarnessModelDisplayIDs
)

// --- Types ---

type GouDemoSystemOpts = sysprompt.GouDemoSystemOpts
type SimpleEnvInfoInput = sysprompt.SimpleEnvInfoInput
type CachedMicrocompactFRCConfig = sysprompt.CachedMicrocompactFRCConfig

// --- Functions: prompts.go ---

var (
	SleepToolPrompt          = sysprompt.SleepToolPrompt
	PrependBullets           = sysprompt.PrependBullets
	SystemRemindersSection   = sysprompt.SystemRemindersSection
	HooksSection             = sysprompt.HooksSection
	SimpleSystemSection      = sysprompt.SimpleSystemSection
	DoingTasksSection        = sysprompt.DoingTasksSection
	ActionsSection           = sysprompt.ActionsSection
	ShouldUseGlobalCacheScope = sysprompt.ShouldUseGlobalCacheScope
	ComputeSimpleEnvInfo     = sysprompt.ComputeSimpleEnvInfo
	GetScratchpadInstructions = sysprompt.GetScratchpadInstructions
	NumericLengthAnchorsSection = sysprompt.NumericLengthAnchorsSection
	TokenBudgetSection        = sysprompt.TokenBudgetSection
	DiscoverSkillsGuidance    = sysprompt.DiscoverSkillsGuidance
	SimpleModeSystemPrompt    = sysprompt.SimpleModeSystemPrompt
	PromptGitHints            = sysprompt.PromptGitHints
	ProactiveSystemPromptParts     = sysprompt.ProactiveSystemPromptParts
	ProactiveAutonomousWorkSection = sysprompt.ProactiveAutonomousWorkSection
)

// --- Functions: gou_demo_system.go ---

var (
	GouDemoReplModeFromEnv        = sysprompt.GouDemoReplModeFromEnv
	GouDemoExplorePlanAgentsFromEnv = sysprompt.GouDemoExplorePlanAgentsFromEnv
	ApplyGouDemoRuntimeEnv        = sysprompt.ApplyGouDemoRuntimeEnv
	BuildGouDemoSystemPrompt      = sysprompt.BuildGouDemoSystemPrompt
	LanguageSection               = sysprompt.LanguageSection
	OutputStyleSection            = sysprompt.OutputStyleSection
)

// --- Functions: session_guidance.go ---

var (
	SessionSpecificGuidance     = sysprompt.SessionSpecificGuidance
	SessionSpecificGuidanceFull = sysprompt.SessionSpecificGuidanceFull
	EnabledToolNames            = sysprompt.EnabledToolNames
)

// --- Functions: prompts_gates.go ---

var (
	BriefProactiveSectionBody             = sysprompt.BriefProactiveSectionBody
	BriefEntitled                          = sysprompt.BriefEntitled
	BriefEnabled                           = sysprompt.BriefEnabled
	GetBriefSection                        = sysprompt.GetBriefSection
	IsMcpInstructionsDeltaEnabled          = sysprompt.IsMcpInstructionsDeltaEnabled
	ProactiveModeActive                    = sysprompt.ProactiveModeActive
	ForkSubagentEnabled                    = sysprompt.ForkSubagentEnabled
	CachedMicrocompactFRCFromEnv           = sysprompt.CachedMicrocompactFRCFromEnv
	FunctionResultClearingSection           = sysprompt.FunctionResultClearingSection
	AntModelDefaultSystemPromptSuffixFromEnv = sysprompt.AntModelDefaultSystemPromptSuffixFromEnv
)

// --- Functions: memory_auto_prompt.go ---

var BuildAutoMemoryPrompt = sysprompt.BuildAutoMemoryPrompt
