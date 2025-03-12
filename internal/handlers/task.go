package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mdayat/demi-masa-backend-service/configs"
	"github.com/mdayat/demi-masa-backend-service/internal/httputil"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
	"github.com/mdayat/demi-masa-backend-service/internal/services"
	"github.com/mdayat/demi-masa-backend-service/repository"
	"github.com/rs/zerolog/log"
)

type TaskHandler interface {
	GetTasks(res http.ResponseWriter, req *http.Request)
	CreateTask(res http.ResponseWriter, req *http.Request)
	UpdateTask(res http.ResponseWriter, req *http.Request)
	DeleteTask(res http.ResponseWriter, req *http.Request)
}

type task struct {
	configs configs.Configs
	service services.TaskServicer
}

func NewTaskHandler(configs configs.Configs, service services.TaskServicer) TaskHandler {
	return &task{
		configs: configs,
		service: service,
	}
}

type getTaskResponse struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Checked     bool   `json:"checked"`
}

func (t task) GetTasks(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	userId := ctx.Value(userIdKey{}).(string)
	tasks, err := retryutil.RetryWithData(func() ([]repository.Task, error) {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return []repository.Task{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return t.configs.Db.Queries.SelectTasksByUserId(ctx, pgtype.UUID{Bytes: userUUID, Valid: true})
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select tasks by user Id")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := make([]getTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		resBody = append(resBody, getTaskResponse{
			Id:          task.ID.String(),
			Name:        task.Name,
			Description: task.Description,
			Checked:     task.Checked,
		})
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusOK,
		ResBody:    resBody,
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully got tasks")
}

func (t task) CreateTask(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		Name        string `json:"name" validate:"required"`
		Description string `json:"description" validate:"required"`
	}

	if err := httputil.DecodeAndValidate(req, t.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	taskUUID := uuid.New()
	userId := ctx.Value(userIdKey{}).(string)

	task, err := retryutil.RetryWithData(func() (repository.Task, error) {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return repository.Task{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return t.configs.Db.Queries.InsertTask(ctx, repository.InsertTaskParams{
			ID:          pgtype.UUID{Bytes: taskUUID, Valid: true},
			UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
			Name:        reqBody.Name,
			Description: reqBody.Description,
		})
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to insert task")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := getTaskResponse{
		Id:          task.ID.String(),
		Name:        task.Name,
		Description: task.Description,
		Checked:     task.Checked,
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusCreated,
		ResBody:    resBody,
	}

	res.Header().Set("Location", fmt.Sprintf("%s/tasks/%s", t.configs.Env.OriginURL, resBody.Id))
	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusCreated).Msg("successfully created task")
}

func (t task) UpdateTask(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	var reqBody struct {
		Name        string `json:"name" validate:"required"`
		Description string `json:"description" validate:"required"`
		Checked     bool   `json:"checked" validate:"required"`
	}

	if err := httputil.DecodeAndValidate(req, t.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	taskId := chi.URLParam(req, "taskId")
	userId := ctx.Value(userIdKey{}).(string)

	task, err := retryutil.RetryWithData(func() (repository.Task, error) {
		taskUUID, err := uuid.Parse(taskId)
		if err != nil {
			return repository.Task{}, fmt.Errorf("failed to parse task Id to UUID: %w", err)
		}

		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return repository.Task{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return t.configs.Db.Queries.UpdateTaskById(ctx, repository.UpdateTaskByIdParams{
			ID:          pgtype.UUID{Bytes: taskUUID, Valid: true},
			UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
			Name:        reqBody.Name,
			Description: reqBody.Description,
			Checked:     reqBody.Checked,
		})
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("task not found")
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to update task")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	resBody := getTaskResponse{
		Id:          task.ID.String(),
		Name:        task.Name,
		Description: task.Description,
		Checked:     task.Checked,
	}

	params := httputil.SendSuccessResponseParams{
		StatusCode: http.StatusOK,
		ResBody:    resBody,
	}

	if err := httputil.SendSuccessResponse(res, params); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to send success response")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully updated task")
}

func (t task) DeleteTask(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logger := log.Ctx(ctx).With().Logger()

	taskId := chi.URLParam(req, "taskId")
	userId := ctx.Value(userIdKey{}).(string)

	err := retryutil.RetryWithoutData(func() error {
		taskUUID, err := uuid.Parse(taskId)
		if err != nil {
			return fmt.Errorf("failed to parse task Id to UUID: %w", err)
		}

		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return t.configs.Db.Queries.DeleteTaskById(ctx, repository.DeleteTaskByIdParams{
			ID:     pgtype.UUID{Bytes: taskUUID, Valid: true},
			UserID: pgtype.UUID{Bytes: userUUID, Valid: true},
		})
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("task not found")
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to delete task")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	logger.Info().Int("status_code", http.StatusOK).Msg("successfully deleted task")
}
