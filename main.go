package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// ScheduleRequest represents the incoming request format
type ScheduleRequest struct {
	ScheduledAt string      `json:"scheduled_at"`
	Endpoint    string      `json:"endpoint"`
	Payload     interface{} `json:"payload"`
	ID          string      `json:"id,omitempty"` // Added ID field for task identification
}

// TaskStore for our scheduled tasks
type TaskStore struct {
	tasks map[string][]ScheduleRequest
	mutex sync.RWMutex
}

// Global task store
var taskStore = &TaskStore{
	tasks: make(map[string][]ScheduleRequest),
}

// Adds a task to the store
func (ts *TaskStore) AddTask(task ScheduleRequest) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	ts.tasks[task.ScheduledAt] = append(ts.tasks[task.ScheduledAt], task)
}

// Removes a task from the store
func (ts *TaskStore) RemoveTask(scheduledAt string, taskIndex int) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	// Check if the scheduled time exists and the index is valid
	if tasks, exists := ts.tasks[scheduledAt]; exists && taskIndex < len(tasks) {
		// Remove the task at the specified index
		ts.tasks[scheduledAt] = append(tasks[:taskIndex], tasks[taskIndex+1:]...)

		// If no more tasks at this time, remove the time entry
		if len(ts.tasks[scheduledAt]) == 0 {
			delete(ts.tasks, scheduledAt)
		}
	}
}

// GetAllTasks returns all scheduled tasks in a formatted way
func (ts *TaskStore) GetAllTasks() []ScheduleRequest {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	var allTasks []ScheduleRequest

	// Collect all tasks from all time slots
	for _, tasks := range ts.tasks {
		allTasks = append(allTasks, tasks...)
	}

	return allTasks
}

// Main handler function for scheduling tasks
func scheduleHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var scheduleReq ScheduleRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&scheduleReq); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate the required fields
	if scheduleReq.Endpoint == "" {
		http.Error(w, "Endpoint is required", http.StatusBadRequest)
		return
	}

	if scheduleReq.ScheduledAt == "" {
		http.Error(w, "scheduled_at is required", http.StatusBadRequest)
		return
	}

	// Parse the scheduled time
	scheduledTime, err := time.Parse(time.RFC3339, scheduleReq.ScheduledAt)
	if err != nil {
		http.Error(w, "Invalid date format. Use RFC3339 format (e.g. 2025-03-10T15:04:05Z)", http.StatusBadRequest)
		return
	}

	// Check if the scheduled time is in the future
	if scheduledTime.Before(time.Now()) {
		http.Error(w, "Scheduled time must be in the future", http.StatusBadRequest)
		return
	}

	// Generate a unique ID for the task if not provided
	if scheduleReq.ID == "" {
		scheduleReq.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	// Add the task to our store
	taskStore.AddTask(scheduleReq)

	// Schedule the task to be executed at the specified time
	go scheduleTask(scheduleReq, scheduledTime)

	// Return success response
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "scheduled",
		"id":      scheduleReq.ID,
		"message": fmt.Sprintf("Task scheduled to run at %s", scheduledTime.Format(time.RFC3339)),
	})
}

// Function to execute the task at the scheduled time
func scheduleTask(task ScheduleRequest, scheduledTime time.Time) {
	// Using time.Until instead of scheduledTime.Sub(time.Now())
	duration := time.Until(scheduledTime)

	// Create a timer for the task
	timer := time.NewTimer(duration)

	// Wait until the timer expires
	<-timer.C

	// Execute the task
	executeTask(task)

	// Remove the task from the store after execution
	removeExecutedTask(task)
}

// Remove a task from the store after execution
func removeExecutedTask(task ScheduleRequest) {
	// Find and remove the executed task
	taskStore.mutex.RLock()
	tasks, exists := taskStore.tasks[task.ScheduledAt]
	if !exists {
		taskStore.mutex.RUnlock()
		return
	}

	// Find the index of the task
	taskIndex := -1
	for i, t := range tasks {
		if t.ID == task.ID {
			taskIndex = i
			break
		}
	}
	taskStore.mutex.RUnlock()

	// If found, remove it
	if taskIndex >= 0 {
		taskStore.RemoveTask(task.ScheduledAt, taskIndex)
		log.Printf("Task %s removed from queue after execution", task.ID)
	}
}

// Execute the scheduled task by making a POST request
func executeTask(task ScheduleRequest) {
	// Convert payload back to JSON
	payload, err := json.Marshal(task.Payload)
	if err != nil {
		log.Printf("Error marshalling payload: %v", err)
		return
	}

	// Create the request with the payload in the body
	req, err := http.NewRequest(http.MethodPost, task.Endpoint, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error executing scheduled task: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Task executed for endpoint %s with status code %d", task.Endpoint, resp.StatusCode)
}

// Updated function to properly format the scheduled tasks
func scheduleView(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all scheduled tasks
	tasks := taskStore.GetAllTasks()

	// Create a more user-friendly response structure
	type TaskResponse struct {
		TotalTasks int               `json:"total_tasks"`
		Tasks      []ScheduleRequest `json:"tasks"`
	}

	response := TaskResponse{
		TotalTasks: len(tasks),
		Tasks:      tasks,
	}

	// Convert to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error retrieving scheduled tasks", http.StatusInternalServerError)
		return
	}

	// Set content type to JSON and return the tasks
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

func main() {
	// Set up the handler for the schedule endpoint
	http.HandleFunc("/schedule", scheduleHandler)
	http.HandleFunc("/schedule-view", scheduleView)

	// Start the server on port 8080
	port := ":8080"
	fmt.Printf("Starting scheduler server on port %s...\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
