package compactservice

import (
	"regexp"
	"strings"
)

// PartialCompactDirection mirrors src/types/message.ts PartialCompactDirection.
type PartialCompactDirection string

const (
	PartialCompactDirectionFrom PartialCompactDirection = "from"
	PartialCompactDirectionUpTo PartialCompactDirection = "up_to"
)

// noToolsPreamble mirrors NO_TOOLS_PREAMBLE in claude-code/src/services/compact/prompt.ts.
// Kept verbatim to preserve wire-level cache key parity with TS.
const noToolsPreamble = `CRITICAL: Respond with TEXT ONLY. Do NOT call any tools.

- Do NOT use Read, Bash, Grep, Glob, Edit, Write, or ANY other tool.
- You already have all the context you need in the conversation above.
- Tool calls will be REJECTED and will waste your only turn — you will fail the task.
- Your entire response must be plain text: an <analysis> block followed by a <summary> block.

`

const detailedAnalysisInstructionBase = `Before providing your final summary, wrap your analysis in <analysis> tags to organize your thoughts and ensure you've covered all necessary points. In your analysis process:

1. Chronologically analyze each message and section of the conversation. For each section thoroughly identify:
   - The user's explicit requests and intents
   - Your approach to addressing the user's requests
   - Key decisions, technical concepts and code patterns
   - Specific details like:
     - file names
     - full code snippets
     - function signatures
     - file edits
   - Errors that you ran into and how you fixed them
   - Pay special attention to specific user feedback that you received, especially if the user told you to do something differently.
2. Double-check for technical accuracy and completeness, addressing each required element thoroughly.`

const detailedAnalysisInstructionPartial = `Before providing your final summary, wrap your analysis in <analysis> tags to organize your thoughts and ensure you've covered all necessary points. In your analysis process:

1. Analyze the recent messages chronologically. For each section thoroughly identify:
   - The user's explicit requests and intents
   - Your approach to addressing the user's requests
   - Key decisions, technical concepts and code patterns
   - Specific details like:
     - file names
     - full code snippets
     - function signatures
     - file edits
   - Errors that you ran into and how you fixed them
   - Pay special attention to specific user feedback that you received, especially if the user told you to do something differently.
2. Double-check for technical accuracy and completeness, addressing each required element thoroughly.`

const baseCompactPromptHead = `Your task is to create a detailed summary of the conversation so far, paying close attention to the user's explicit requests and your previous actions.
This summary should be thorough in capturing technical details, code patterns, and architectural decisions that would be essential for continuing development work without losing context.

`

const baseCompactPromptBody = `

Your summary should include the following sections:

1. Primary Request and Intent: Capture all of the user's explicit requests and intents in detail
2. Key Technical Concepts: List all important technical concepts, technologies, and frameworks discussed.
3. Files and Code Sections: Enumerate specific files and code sections examined, modified, or created. Pay special attention to the most recent messages and include full code snippets where applicable and include a summary of why this file read or edit is important.
4. Errors and fixes: List all errors that you ran into, and how you fixed them. Pay special attention to specific user feedback that you received, especially if the user told you to do something differently.
5. Problem Solving: Document problems solved and any ongoing troubleshooting efforts.
6. All user messages: List ALL user messages that are not tool results. These are critical for understanding the users' feedback and changing intent.
7. Pending Tasks: Outline any pending tasks that you have explicitly been asked to work on.
8. Current Work: Describe in detail precisely what was being worked on immediately before this summary request, paying special attention to the most recent messages from both user and assistant. Include file names and code snippets where applicable.
9. Optional Next Step: List the next step that you will take that is related to the most recent work you were doing. IMPORTANT: ensure that this step is DIRECTLY in line with the user's most recent explicit requests, and the task you were working on immediately before this summary request. If your last task was concluded, then only list next steps if they are explicitly in line with the users request. Do not start on tangential requests or really old requests that were already completed without confirming with the user first.
                       If there is a next step, include direct quotes from the most recent conversation showing exactly what task you were working on and where you left off. This should be verbatim to ensure there's no drift in task interpretation.

Here's an example of how your output should be structured:

<example>
<analysis>
[Your thought process, ensuring all points are covered thoroughly and accurately]
</analysis>

<summary>
1. Primary Request and Intent:
   [Detailed description]

2. Key Technical Concepts:
   - [Concept 1]
   - [Concept 2]
   - [...]

3. Files and Code Sections:
   - [File Name 1]
      - [Summary of why this file is important]
      - [Summary of the changes made to this file, if any]
      - [Important Code Snippet]
   - [File Name 2]
      - [Important Code Snippet]
   - [...]

4. Errors and fixes:
    - [Detailed description of error 1]:
      - [How you fixed the error]
      - [User feedback on the error if any]
    - [...]

5. Problem Solving:
   [Description of solved problems and ongoing troubleshooting]

6. All user messages: 
    - [Detailed non tool use user message]
    - [...]

7. Pending Tasks:
   - [Task 1]
   - [Task 2]
   - [...]

8. Current Work:
   [Precise description of current work]

9. Optional Next Step:
   [Optional Next step to take]

</summary>
</example>

Please provide your summary based on the conversation so far, following this structure and ensuring precision and thoroughness in your response. 

There may be additional summarization instructions provided in the included context. If so, remember to follow these instructions when creating the above summary. Examples of instructions include:
<example>
## Compact Instructions
When summarizing the conversation focus on typescript code changes and also remember the mistakes you made and how you fixed them.
</example>

<example>
# Summary instructions
When you are using compact - please focus on test output and code changes. Include file reads verbatim.
</example>
`

const partialCompactPromptHead = `Your task is to create a detailed summary of the RECENT portion of the conversation — the messages that follow earlier retained context. The earlier messages are being kept intact and do NOT need to be summarized. Focus your summary on what was discussed, learned, and accomplished in the recent messages only.

`

const partialCompactPromptBody = `

Your summary should include the following sections:

1. Primary Request and Intent: Capture the user's explicit requests and intents from the recent messages
2. Key Technical Concepts: List important technical concepts, technologies, and frameworks discussed recently.
3. Files and Code Sections: Enumerate specific files and code sections examined, modified, or created. Include full code snippets where applicable and include a summary of why this file read or edit is important.
4. Errors and fixes: List errors encountered and how they were fixed.
5. Problem Solving: Document problems solved and any ongoing troubleshooting efforts.
6. All user messages: List ALL user messages from the recent portion that are not tool results.
7. Pending Tasks: Outline any pending tasks from the recent messages.
8. Current Work: Describe precisely what was being worked on immediately before this summary request.
9. Optional Next Step: List the next step related to the most recent work. Include direct quotes from the most recent conversation.

Here's an example of how your output should be structured:

<example>
<analysis>
[Your thought process, ensuring all points are covered thoroughly and accurately]
</analysis>

<summary>
1. Primary Request and Intent:
   [Detailed description]

2. Key Technical Concepts:
   - [Concept 1]
   - [Concept 2]

3. Files and Code Sections:
   - [File Name 1]
      - [Summary of why this file is important]
      - [Important Code Snippet]

4. Errors and fixes:
    - [Error description]:
      - [How you fixed it]

5. Problem Solving:
   [Description]

6. All user messages:
    - [Detailed non tool use user message]

7. Pending Tasks:
   - [Task 1]

8. Current Work:
   [Precise description of current work]

9. Optional Next Step:
   [Optional Next step to take]

</summary>
</example>

Please provide your summary based on the RECENT messages only (after the retained earlier context), following this structure and ensuring precision and thoroughness in your response.
`

const partialCompactUpToPromptHead = `Your task is to create a detailed summary of this conversation. This summary will be placed at the start of a continuing session; newer messages that build on this context will follow after your summary (you do not see them here). Summarize thoroughly so that someone reading only your summary and then the newer messages can fully understand what happened and continue the work.

`

const partialCompactUpToPromptBody = `

Your summary should include the following sections:

1. Primary Request and Intent: Capture the user's explicit requests and intents in detail
2. Key Technical Concepts: List important technical concepts, technologies, and frameworks discussed.
3. Files and Code Sections: Enumerate specific files and code sections examined, modified, or created. Include full code snippets where applicable and include a summary of why this file read or edit is important.
4. Errors and fixes: List errors encountered and how they were fixed.
5. Problem Solving: Document problems solved and any ongoing troubleshooting efforts.
6. All user messages: List ALL user messages that are not tool results.
7. Pending Tasks: Outline any pending tasks.
8. Work Completed: Describe what was accomplished by the end of this portion.
9. Context for Continuing Work: Summarize any context, decisions, or state that would be needed to understand and continue the work in subsequent messages.

Here's an example of how your output should be structured:

<example>
<analysis>
[Your thought process, ensuring all points are covered thoroughly and accurately]
</analysis>

<summary>
1. Primary Request and Intent:
   [Detailed description]

2. Key Technical Concepts:
   - [Concept 1]
   - [Concept 2]

3. Files and Code Sections:
   - [File Name 1]
      - [Summary of why this file is important]
      - [Important Code Snippet]

4. Errors and fixes:
    - [Error description]:
      - [How you fixed it]

5. Problem Solving:
   [Description]

6. All user messages:
    - [Detailed non tool use user message]

7. Pending Tasks:
   - [Task 1]

8. Work Completed:
   [Description of what was accomplished]

9. Context for Continuing Work:
   [Key context, decisions, or state needed to continue the work]

</summary>
</example>

Please provide your summary following this structure, ensuring precision and thoroughness in your response.
`

// noToolsTrailer mirrors NO_TOOLS_TRAILER in TS.
const noToolsTrailer = "\n\nREMINDER: Do NOT call any tools. Respond with plain text only — " +
	"an <analysis> block followed by a <summary> block. " +
	"Tool calls will be rejected and you will fail the task."

// GetCompactPrompt mirrors getCompactPrompt(customInstructions?) in TS.
func GetCompactPrompt(customInstructions string) string {
	prompt := noToolsPreamble + baseCompactPromptHead + detailedAnalysisInstructionBase + baseCompactPromptBody
	if strings.TrimSpace(customInstructions) != "" {
		prompt += "\n\nAdditional Instructions:\n" + customInstructions
	}
	prompt += noToolsTrailer
	return prompt
}

// GetPartialCompactPrompt mirrors getPartialCompactPrompt(customInstructions?, direction?) in TS.
func GetPartialCompactPrompt(customInstructions string, direction PartialCompactDirection) string {
	var template string
	if direction == PartialCompactDirectionUpTo {
		template = partialCompactUpToPromptHead + detailedAnalysisInstructionBase + partialCompactUpToPromptBody
	} else {
		template = partialCompactPromptHead + detailedAnalysisInstructionPartial + partialCompactPromptBody
	}
	prompt := noToolsPreamble + template
	if strings.TrimSpace(customInstructions) != "" {
		prompt += "\n\nAdditional Instructions:\n" + customInstructions
	}
	prompt += noToolsTrailer
	return prompt
}

// analysisTagRE matches <analysis>…</analysis> (non-greedy, DOTALL).
var analysisTagRE = regexp.MustCompile(`(?s)<analysis>.*?</analysis>`)

// summaryTagRE captures the contents of <summary>…</summary>.
var summaryTagRE = regexp.MustCompile(`(?s)<summary>(.*?)</summary>`)

// doubleBlankRE collapses 3+ blank lines to exactly 2 ("\n\n+" → "\n\n") like TS.
var doubleBlankRE = regexp.MustCompile(`\n\n+`)

// FormatCompactSummary mirrors formatCompactSummary in TS:
// strips the <analysis> drafting scratchpad and rewrites <summary>…</summary>
// to "Summary:\n…".
func FormatCompactSummary(summary string) string {
	// Strip <analysis>…</analysis> (first match, like String.prototype.replace).
	formatted := analysisTagRE.ReplaceAllString(summary, "")
	// Replace <summary>…</summary> with "Summary:\n<content trimmed>".
	if m := summaryTagRE.FindStringSubmatch(formatted); m != nil {
		content := strings.TrimSpace(m[1])
		replacement := "Summary:\n" + content
		formatted = summaryTagRE.ReplaceAllString(formatted, replacement)
	}
	// Collapse extra blank lines.
	formatted = doubleBlankRE.ReplaceAllString(formatted, "\n\n")
	return strings.TrimSpace(formatted)
}

// CompactUserSummaryOpts groups optional parameters for GetCompactUserSummaryMessage
// to mirror TS' JS-style optional positional params.
type CompactUserSummaryOpts struct {
	SuppressFollowUpQuestions bool
	TranscriptPath            string
	RecentMessagesPreserved   bool
	// ProactiveActive mirrors the TS proactive/isProactiveActive() branch.
	// Default false for external Go builds; hosts that implement proactive mode
	// set this to true to get the autonomous-continuation continuation tail.
	ProactiveActive bool
}

// GetCompactUserSummaryMessage mirrors getCompactUserSummaryMessage in TS.
func GetCompactUserSummaryMessage(summary string, opts CompactUserSummaryOpts) string {
	formatted := FormatCompactSummary(summary)

	base := "This session is being continued from a previous conversation that ran out of context. The summary below covers the earlier portion of the conversation.\n\n" + formatted

	if strings.TrimSpace(opts.TranscriptPath) != "" {
		base += "\n\nIf you need specific details from before compaction (like exact code snippets, error messages, or content you generated), read the full transcript at: " + opts.TranscriptPath
	}

	if opts.RecentMessagesPreserved {
		base += "\n\nRecent messages are preserved verbatim."
	}

	if opts.SuppressFollowUpQuestions {
		continuation := base + "\nContinue the conversation from where it left off without asking the user any further questions. Resume directly — do not acknowledge the summary, do not recap what was happening, do not preface with \"I'll continue\" or similar. Pick up the last task as if the break never happened."
		if opts.ProactiveActive {
			continuation += "\n\nYou are running in autonomous/proactive mode. This is NOT a first wake-up — you were already working autonomously before compaction. Continue your work loop: pick up where you left off based on the summary above. Do not greet the user or ask what to work on."
		}
		return continuation
	}

	return base
}
