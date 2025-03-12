# Task Scheduler API

## Overview
This is a simple HTTP-based task scheduler written in Go. It allows users to schedule tasks that will execute at a future time by making HTTP POST requests to specified endpoints. The tasks are stored in memory and executed asynchronously.

## Features
- Schedule HTTP POST requests to be executed at a specific future time.
- List all scheduled tasks.
- Automatically removes executed tasks from the queue.
- Thread-safe task storage using mutex locks.
- Uses `time.Timer` for precise execution timing.

## Installation & Running the Server
### Prerequisites
- Go 1.18+

### Steps
1. Clone the repository:
   ```sh
   git clone https://github.com/yourusername/task-scheduler.git
   cd task-scheduler
   ```
2. Build and run the server:
   ```sh
   go run main.go
   ```
3. The server starts on port `8080`.

## API Endpoints

### 1. Schedule a Task
**Endpoint:** `POST /schedule`

**Request Body:**
```json
{
  "scheduled_at": "2025-03-10T15:04:05Z",
  "endpoint": "http://example.com/webhook",
  "payload": { "key": "value" }
}
```

**Response:**
```json
{
  "status": "scheduled",
  "id": "task_1712030305000000",
  "message": "Task scheduled to run at 2025-03-10T15:04:05Z"
}
```

### 2. View Scheduled Tasks
**Endpoint:** `GET /schedule-view`

**Response:**
```json
{
  "total_tasks": 1,
  "tasks": [
    {
      "scheduled_at": "2025-03-10T15:04:05Z",
      "endpoint": "http://example.com/webhook",
      "payload": { "key": "value" },
      "id": "task_1712030305000000"
    }
  ]
}
```

## How It Works
1. When a task is scheduled, it's stored in memory along with its execution time.
2. A goroutine starts a timer that waits until the scheduled time.
3. Once the timer expires, an HTTP POST request is sent to the specified endpoint with the provided payload.
4. The task is removed from the store after execution.

## Limitations
- Tasks are stored in memory, so they will be lost if the server restarts.
- No persistence mechanism (e.g., database) is implemented.

## Future Enhancements
- Implement database storage for task persistence.
- Add support for retries on failed task execution.
- Provide an admin dashboard for managing scheduled tasks.

## License
This project is open-source and available under the [MIT License](LICENSE).

## Contributing
Contributions are welcome! Feel free to submit issues and pull requests.

---
Made with ❤️ in Go.
