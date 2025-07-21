package application

// Todo represents a single to-do item.
type Todo struct {
	ID          int    `json:"id"`
	ListID      int    `json:"listId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// TodoList represents a list of to-do items.
type TodoList struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Todos []Todo `json:"todos,omitempty"`
}
