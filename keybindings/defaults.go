package keybindings

// DefaultBindings provides the default keybinding configuration
// This mirrors the DEFAULT_BINDINGS from TypeScript
var DefaultBindings = []KeybindingBlock{
	{
		Context: ContextGlobal,
		Bindings: map[string]*KeybindingActionPtr{
			"ctrl+c":       {Action: stringPtr("app:interrupt")},
			"ctrl+d":       {Action: stringPtr("app:exit")},
			"ctrl+t":       {Action: stringPtr("app:toggleTodos")},
			"ctrl+o":       {Action: stringPtr("app:toggleTranscript")},
			"ctrl+shift+o": {Action: stringPtr("app:toggleTeammatePreview")},
			"ctrl+l":       {Action: stringPtr("app:redraw")},
			"ctrl+r":       {Action: stringPtr("history:search")},
		},
	},
	{
		Context: ContextChat,
		Bindings: map[string]*KeybindingActionPtr{
			"up":              {Action: stringPtr("history:previous")},
			"down":            {Action: stringPtr("history:next")},
			"escape":          {Action: stringPtr("chat:cancel")},
			"ctrl+x ctrl+k":   {Action: stringPtr("chat:killAgents")},
			"shift+tab":       {Action: stringPtr("chat:cycleMode")},
			"meta+p":          {Action: stringPtr("chat:modelPicker")},
			"meta+o":          {Action: stringPtr("chat:fastMode")},
			"meta+t":          {Action: stringPtr("chat:thinkingToggle")},
			"enter":           {Action: stringPtr("chat:submit")},
			"ctrl+_":          {Action: stringPtr("chat:undo")},
			"ctrl+shift+-":    {Action: stringPtr("chat:undo")},
			"ctrl+x ctrl+e":   {Action: stringPtr("chat:externalEditor")},
			"ctrl+g":          {Action: stringPtr("chat:externalEditor")},
			"ctrl+s":          {Action: stringPtr("chat:stash")},
			"ctrl+v":          {Action: stringPtr("chat:imagePaste")},
		},
	},
	{
		Context: ContextAutocomplete,
		Bindings: map[string]*KeybindingActionPtr{
			"tab":    {Action: stringPtr("autocomplete:accept")},
			"escape": {Action: stringPtr("autocomplete:dismiss")},
			"up":     {Action: stringPtr("autocomplete:previous")},
			"down":   {Action: stringPtr("autocomplete:next")},
		},
	},
	{
		Context: ContextConfirmation,
		Bindings: map[string]*KeybindingActionPtr{
			"y":           {Action: stringPtr("confirm:yes")},
			"enter":       {Action: stringPtr("confirm:yes")},
			"escape":      {Action: stringPtr("confirm:no")},
			"n":           {Action: stringPtr("confirm:no")},
			"up":          {Action: stringPtr("confirm:previous")},
			"down":        {Action: stringPtr("confirm:next")},
			"tab":         {Action: stringPtr("confirm:nextField")},
			"shift+tab":   {Action: stringPtr("confirm:cycleMode")},
			"space":       {Action: stringPtr("confirm:toggle")},
			"ctrl+e":      {Action: stringPtr("confirm:toggleExplanation")},
			"ctrl+d":      {Action: stringPtr("permission:toggleDebug")},
		},
	},
	{
		Context: ContextTabs,
		Bindings: map[string]*KeybindingActionPtr{
			"tab":       {Action: stringPtr("tabs:next")},
			"right":     {Action: stringPtr("tabs:next")},
			"shift+tab": {Action: stringPtr("tabs:previous")},
			"left":      {Action: stringPtr("tabs:previous")},
		},
	},
	{
		Context: ContextTranscript,
		Bindings: map[string]*KeybindingActionPtr{
			"ctrl+e": {Action: stringPtr("transcript:toggleShowAll")},
			"ctrl+c": {Action: stringPtr("transcript:exit")},
			"escape": {Action: stringPtr("transcript:exit")},
			"q":      {Action: stringPtr("transcript:exit")},
		},
	},
	{
		Context: ContextHistorySearch,
		Bindings: map[string]*KeybindingActionPtr{
			"ctrl+r": {Action: stringPtr("historySearch:next")},
			"escape": {Action: stringPtr("historySearch:accept")},
			"tab":    {Action: stringPtr("historySearch:accept")},
			"ctrl+c": {Action: stringPtr("historySearch:cancel")},
			"enter":  {Action: stringPtr("historySearch:execute")},
		},
	},
	{
		Context: ContextTask,
		Bindings: map[string]*KeybindingActionPtr{
			"ctrl+b": {Action: stringPtr("task:background")},
		},
	},
	{
		Context: ContextThemePicker,
		Bindings: map[string]*KeybindingActionPtr{
			"ctrl+t": {Action: stringPtr("theme:toggleSyntaxHighlighting")},
		},
	},
	{
		Context: ContextHelp,
		Bindings: map[string]*KeybindingActionPtr{
			"escape": {Action: stringPtr("help:dismiss")},
		},
	},
	{
		Context: ContextAttachments,
		Bindings: map[string]*KeybindingActionPtr{
			"right":     {Action: stringPtr("attachments:next")},
			"left":      {Action: stringPtr("attachments:previous")},
			"backspace": {Action: stringPtr("attachments:remove")},
			"delete":    {Action: stringPtr("attachments:remove")},
			"down":      {Action: stringPtr("attachments:exit")},
			"escape":    {Action: stringPtr("attachments:exit")},
		},
	},
	{
		Context: ContextFooter,
		Bindings: map[string]*KeybindingActionPtr{
			"up":     {Action: stringPtr("footer:up")},
			"ctrl+p": {Action: stringPtr("footer:up")},
			"down":   {Action: stringPtr("footer:down")},
			"ctrl+n": {Action: stringPtr("footer:down")},
			"right":  {Action: stringPtr("footer:next")},
			"left":   {Action: stringPtr("footer:previous")},
			"enter":  {Action: stringPtr("footer:openSelected")},
			"escape": {Action: stringPtr("footer:clearSelection")},
		},
	},
	{
		Context: ContextMessageSelector,
		Bindings: map[string]*KeybindingActionPtr{
			"up":         {Action: stringPtr("messageSelector:up")},
			"k":          {Action: stringPtr("messageSelector:up")},
			"ctrl+p":     {Action: stringPtr("messageSelector:up")},
			"down":       {Action: stringPtr("messageSelector:down")},
			"j":          {Action: stringPtr("messageSelector:down")},
			"ctrl+n":     {Action: stringPtr("messageSelector:down")},
			"ctrl+up":    {Action: stringPtr("messageSelector:top")},
			"shift+up":   {Action: stringPtr("messageSelector:top")},
			"meta+up":    {Action: stringPtr("messageSelector:top")},
			"shift+k":    {Action: stringPtr("messageSelector:top")},
			"ctrl+down":  {Action: stringPtr("messageSelector:bottom")},
			"shift+down": {Action: stringPtr("messageSelector:bottom")},
			"meta+down":  {Action: stringPtr("messageSelector:bottom")},
			"shift+j":    {Action: stringPtr("messageSelector:bottom")},
			"enter":      {Action: stringPtr("messageSelector:select")},
		},
	},
	{
		Context: ContextDiffDialog,
		Bindings: map[string]*KeybindingActionPtr{
			"escape": {Action: stringPtr("diff:dismiss")},
			"left":   {Action: stringPtr("diff:previousSource")},
			"right":  {Action: stringPtr("diff:nextSource")},
			"enter":  {Action: stringPtr("diff:viewDetails")},
			"up":     {Action: stringPtr("diff:previousFile")},
			"down":   {Action: stringPtr("diff:nextFile")},
		},
	},
	{
		Context: ContextModelPicker,
		Bindings: map[string]*KeybindingActionPtr{
			"left":  {Action: stringPtr("modelPicker:decreaseEffort")},
			"right": {Action: stringPtr("modelPicker:increaseEffort")},
		},
	},
	{
		Context: ContextSettings,
		Bindings: map[string]*KeybindingActionPtr{
			"down":   {Action: stringPtr("select:next")},
			"j":      {Action: stringPtr("select:next")},
			"ctrl+n": {Action: stringPtr("select:next")},
			"up":     {Action: stringPtr("select:previous")},
			"k":      {Action: stringPtr("select:previous")},
			"ctrl+p": {Action: stringPtr("select:previous")},
			"space":  {Action: stringPtr("select:accept")},
			"enter":  {Action: stringPtr("settings:close")},
			"escape": {Action: stringPtr("confirm:no")},
			"/":      {Action: stringPtr("settings:search")},
			"r":      {Action: stringPtr("settings:retry")},
		},
	},
	{
		Context: ContextSelect,
		Bindings: map[string]*KeybindingActionPtr{
			"down":   {Action: stringPtr("select:next")},
			"j":      {Action: stringPtr("select:next")},
			"ctrl+n": {Action: stringPtr("select:next")},
			"up":     {Action: stringPtr("select:previous")},
			"k":      {Action: stringPtr("select:previous")},
			"ctrl+p": {Action: stringPtr("select:previous")},
			"space":  {Action: stringPtr("select:accept")},
			"enter":  {Action: stringPtr("select:accept")},
			"escape": {Action: stringPtr("select:cancel")},
		},
	},
	{
		Context: ContextPlugin,
		Bindings: map[string]*KeybindingActionPtr{
			"space": {Action: stringPtr("plugin:toggle")},
			"i":     {Action: stringPtr("plugin:install")},
		},
	},
}

// stringPtr creates a pointer to a KeybindingAction string
func stringPtr(s string) *KeybindingAction {
	action := KeybindingAction(s)
	return &action
}