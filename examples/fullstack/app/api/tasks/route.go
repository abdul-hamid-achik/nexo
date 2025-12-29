package tasks

import (
	"sync"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

// Task represents a simple task
type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// In-memory task store (for demo purposes)
var (
	tasks   = []Task{}
	tasksMu sync.RWMutex
	nextID  = 1
)

// Get returns the task list as HTML (for HTMX)
func Get(c *fuego.Context) error {
	tasksMu.RLock()
	defer tasksMu.RUnlock()

	// Return HTML for HTMX
	html := ""
	if len(tasks) == 0 {
		html = `<p class="text-gray-500 italic">No tasks yet. Add one above!</p>`
	} else {
		html = `<ul class="space-y-2">`
		for _, task := range tasks {
			checkedClass := ""
			checkedAttr := ""
			if task.Completed {
				checkedClass = " line-through text-gray-400"
				checkedAttr = ` checked`
			}
			html += `<li class="flex items-center gap-3 p-3 bg-gray-50 rounded-lg">
				<input 
					type="checkbox" 
					class="h-4 w-4 text-orange-600 focus:ring-orange-500 border-gray-300 rounded"
					hx-post="/api/tasks/toggle?id=` + itoa(task.ID) + `"
					hx-target="#task-list"
					hx-swap="innerHTML"` + checkedAttr + `
				/>
				<span class="flex-1` + checkedClass + `">` + task.Title + `</span>
				<button 
					class="text-red-500 hover:text-red-700"
					hx-delete="/api/tasks?id=` + itoa(task.ID) + `"
					hx-target="#task-list"
					hx-swap="innerHTML"
				>
					Delete
				</button>
			</li>`
		}
		html += `</ul>`
	}

	return c.HTML(200, html)
}

// Post adds a new task
func Post(c *fuego.Context) error {
	title := c.FormValue("title")
	if title == "" {
		return c.HTML(400, `<p class="text-red-500">Task title is required</p>`)
	}

	tasksMu.Lock()
	task := Task{
		ID:        nextID,
		Title:     title,
		Completed: false,
	}
	tasks = append(tasks, task)
	nextID++
	tasksMu.Unlock()

	// Return the updated task list
	return Get(c)
}

// Delete removes a task
func Delete(c *fuego.Context) error {
	id := c.QueryInt("id", 0)
	if id == 0 {
		return c.HTML(400, `<p class="text-red-500">Task ID is required</p>`)
	}

	tasksMu.Lock()
	for i, task := range tasks {
		if task.ID == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			break
		}
	}
	tasksMu.Unlock()

	// Return the updated task list
	return Get(c)
}

// Helper to convert int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
