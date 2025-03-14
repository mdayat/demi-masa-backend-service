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
	"github.com/mdayat/demi-masa-backend-service/internal/dtos"
	"github.com/mdayat/demi-masa-backend-service/internal/httputil"
	"github.com/mdayat/demi-masa-backend-service/internal/retryutil"
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
}

func NewTaskHandler(configs configs.Configs) TaskHandler {
	return &task{
		configs: configs,
	}
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

		return t.configs.Db.Queries.SelectUserTasks(ctx, pgtype.UUID{Bytes: userUUID, Valid: true})
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to select user tasks")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := make([]dtos.TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		resBody = append(resBody, dtos.TaskResponse{
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

	var reqBody dtos.CreateTaskRequest
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

		return t.configs.Db.Queries.InsertUserTask(ctx, repository.InsertUserTaskParams{
			ID:          pgtype.UUID{Bytes: taskUUID, Valid: true},
			UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
			Name:        reqBody.Name,
			Description: reqBody.Description,
		})
	})

	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to insert user task")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resBody := dtos.TaskResponse{
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

	var reqBody dtos.UpdateTaskRequest
	if err := httputil.DecodeAndValidate(req, t.configs.Validate, &reqBody); err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusBadRequest).Msg("invalid request body")
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	taskId := chi.URLParam(req, "taskId")
	taskUUID, err := uuid.Parse(taskId)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("task not found")
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if reqBody.Name == "" && reqBody.Description == "" && reqBody.Checked == nil {
		res.WriteHeader(http.StatusNoContent)
		logger.Info().Int("status_code", http.StatusNoContent).Msg("no update performed")
		return
	}

	var name pgtype.Text
	if reqBody.Name != "" {
		name = pgtype.Text{String: reqBody.Name, Valid: true}
	}

	var description pgtype.Text
	if reqBody.Description != "" {
		description = pgtype.Text{String: reqBody.Description, Valid: true}
	}

	var checked pgtype.Bool
	if reqBody.Checked != nil {
		checked = pgtype.Bool{Bool: *reqBody.Checked, Valid: true}
	}

	userId := ctx.Value(userIdKey{}).(string)
	task, err := retryutil.RetryWithData(func() (repository.Task, error) {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return repository.Task{}, fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return t.configs.Db.Queries.UpdateUserTask(ctx, repository.UpdateUserTaskParams{
			ID:          pgtype.UUID{Bytes: taskUUID, Valid: true},
			UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
			Name:        name,
			Description: description,
			Checked:     checked,
		})
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("task not found")
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to update user task")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	resBody := dtos.TaskResponse{
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
	taskUUID, err := uuid.Parse(taskId)
	if err != nil {
		logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("task not found")
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	userId := ctx.Value(userIdKey{}).(string)
	err = retryutil.RetryWithoutData(func() error {
		userUUID, err := uuid.Parse(userId)
		if err != nil {
			return fmt.Errorf("failed to parse user Id to UUID: %w", err)
		}

		return t.configs.Db.Queries.DeleteUserTask(ctx, repository.DeleteUserTaskParams{
			ID:     pgtype.UUID{Bytes: taskUUID, Valid: true},
			UserID: pgtype.UUID{Bytes: userUUID, Valid: true},
		})
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusNotFound).Msg("task not found")
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			logger.Error().Err(err).Caller().Int("status_code", http.StatusInternalServerError).Msg("failed to delete user task")
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	res.WriteHeader(http.StatusNoContent)
	logger.Info().Int("status_code", http.StatusNoContent).Msg("successfully deleted task")
}
