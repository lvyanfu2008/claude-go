package appstate

import "encoding/json"

// TodoStatus mirrors src/utils/todo/types.ts todo status enum.
type TodoStatus string

const (
	TodoPending    TodoStatus = "pending"
	TodoInProgress TodoStatus = "in_progress"
	TodoCompleted  TodoStatus = "completed"
)

// TodoItem mirrors src/utils/todo/types.ts TodoItem.
type TodoItem struct {
	Content    string     `json:"content"`
	Status     TodoStatus `json:"status"`
	ActiveForm string     `json:"activeForm"`
}

// TodoList mirrors src/utils/todo/types.ts TodoList (array of items).
type TodoList []TodoItem

// TodosMap mirrors AppState.todos { [agentId: string]: TodoList }.
type TodosMap map[string]TodoList

// MarshalJSON encodes nil map as {} and nil slices per agent as [].
func (m TodosMap) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	out := make(map[string][]TodoItem, len(m))
	for k, v := range m {
		if v == nil {
			out[k] = []TodoItem{}
		} else {
			out[k] = []TodoItem(v)
		}
	}
	return json.Marshal(out)
}

// UnmarshalJSON normalizes nil slices for each agent id.
func (m *TodosMap) UnmarshalJSON(b []byte) error {
	var raw map[string][]TodoItem
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw == nil {
		*m = TodosMap{}
		return nil
	}
	tm := make(TodosMap, len(raw))
	for k, v := range raw {
		if v == nil {
			tm[k] = []TodoItem{}
		} else {
			tm[k] = TodoList(v)
		}
	}
	*m = tm
	return nil
}
