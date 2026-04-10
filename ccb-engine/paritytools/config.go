package paritytools

// Config is passed from [skilltools.ParityToolRunner] into unconditional tool runners.
type Config struct {
	Roots            []string
	WorkDir          string
	ProjectRoot      string
	SessionID    string
	AskAutoFirst bool // when true, AskUserQuestion picks the first option per question (gou-demo default)
}
