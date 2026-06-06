package state

type Modal struct {
	Permission   interface{}
	Question     interface{}
	HooksConfig  interface{}
	AskAutoFirst bool
}
