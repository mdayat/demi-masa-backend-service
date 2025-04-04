package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/mdayat/demi-masa-backend-service/internal/dtos"
)

func TestTaskHandlers(t *testing.T) {
	ctx := context.TODO()
	var createdTask dtos.TaskResponse

	createTaskTable := []struct {
		name           string
		reqBody        string
		expectedStatus int
		expectedResult dtos.TaskResponse
	}{
		{
			name:           "CreateTask/Success",
			reqBody:        `{"name": "name", "description": "description"}`,
			expectedStatus: http.StatusCreated,
			expectedResult: dtos.TaskResponse{
				Name:        "name",
				Description: "description",
				Checked:     false,
			},
		},
		{
			name:           "CreateTask/Bad Request (name)",
			reqBody:        `{"name": "", "description": "description"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "CreateTask/Bad Request (description)",
			reqBody:        `{"name": "name", "description": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, v := range createTaskTable {
		t.Run(v.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/tasks", testServer.URL)
			res, err := testClient.Post(url, "application/json", bytes.NewBuffer([]byte(v.reqBody)))
			if err != nil {
				t.Fatalf("wasn't expecting error, got: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != v.expectedStatus {
				t.Fatalf("expected status %d, got %d", v.expectedStatus, res.StatusCode)
			}

			if v.expectedStatus == http.StatusCreated {
				if err := json.NewDecoder(res.Body).Decode(&createdTask); err != nil {
					t.Fatalf("unexpected response body: %v", res)
				}

				if diff := cmp.Diff(v.expectedResult, createdTask, cmpopts.IgnoreFields(dtos.TaskResponse{}, "Id")); diff != "" {
					t.Error(diff)
				}
			}
		})
	}

	t.Run("GetTasks/Success", func(t *testing.T) {
		res, err := testClient.Get(fmt.Sprintf("%s/tasks", testServer.URL))
		if err != nil {
			t.Fatalf("wasn't expecting error, got: %v", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
		}

		var tasks []dtos.TaskResponse
		if err = json.NewDecoder(res.Body).Decode(&tasks); err != nil {
			t.Fatalf("unexpected response body: %v", res)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 task, got %d", len(tasks))
		}

		for _, task := range tasks {
			if task.Id != createdTask.Id {
				continue
			}

			if diff := cmp.Diff(createdTask, task); diff != "" {
				t.Error(diff)
			}
		}
	})

	updateTaskTable := []struct {
		name           string
		taskId         string
		reqBody        string
		expectedStatus int
		expectedResult dtos.TaskResponse
	}{
		{
			name:           "UpdateTask/Success",
			taskId:         createdTask.Id,
			reqBody:        `{"name": "name changed"}`,
			expectedStatus: http.StatusOK,
			expectedResult: dtos.TaskResponse{
				Id:          createdTask.Id,
				Name:        "name changed",
				Description: createdTask.Description,
				Checked:     createdTask.Checked,
			},
		},
		{
			name:           "UpdateTask/Success (no update performed)",
			taskId:         createdTask.Id,
			reqBody:        `{}`,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "UpdateTask/Bad Request (name)",
			taskId:         createdTask.Id,
			reqBody:        `{"name": 1}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "UpdateTask/Bad Request (description)",
			taskId:         createdTask.Id,
			reqBody:        `{"description": 1}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "UpdateTask/Not Found",
			taskId:         uuid.NewString(),
			reqBody:        `{"name": "name changed"}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, v := range updateTaskTable {
		t.Run(v.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/tasks/%s", testServer.URL, v.taskId)
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer([]byte(v.reqBody)))
			if err != nil {
				t.Fatalf("wasn't expecting error, got: %v", err)
			}

			res, err := testClient.Do(req)
			if err != nil {
				t.Fatalf("wasn't expecting error, got: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != v.expectedStatus {
				t.Fatalf("expected status %d, got %d", v.expectedStatus, res.StatusCode)
			}

			if v.expectedStatus == http.StatusOK {
				var updatedTask dtos.TaskResponse
				if err := json.NewDecoder(res.Body).Decode(&updatedTask); err != nil {
					t.Fatalf("unexpected response body: %v", res)
				}

				if diff := cmp.Diff(v.expectedResult, updatedTask); diff != "" {
					t.Error(diff)
				}
			}
		})
	}

	deleteTaskTable := []struct {
		name           string
		taskId         string
		expectedStatus int
	}{
		{
			name:           "DeleteTask/Success",
			taskId:         createdTask.Id,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "DeleteTask/Not Found",
			taskId:         uuid.NewString(),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, v := range deleteTaskTable {
		t.Run(v.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/tasks/%s", testServer.URL, v.taskId)
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
			if err != nil {
				t.Fatalf("wasn't expecting error, got: %v", err)
			}

			res, err := testClient.Do(req)
			if err != nil {
				t.Fatalf("wasn't expecting error, got: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != v.expectedStatus {
				t.Errorf("expected status %d, got %d", v.expectedStatus, res.StatusCode)
			}
		})
	}
}
