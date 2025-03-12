package services

import (
	"github.com/mdayat/demi-masa-backend-service/configs"
)

type TaskServicer interface{}

type task struct {
	configs configs.Configs
}

func NewTaskService(configs configs.Configs) TaskServicer {
	return &task{
		configs: configs,
	}
}
