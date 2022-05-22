package tasks

import "github.com/hibiken/asynq"

//GetMux maps names to handlers
func GetMux() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskExtraLifeTeamUpdate, HandleExtraLifeTeamUpdateTask)
	mux.HandleFunc(TaskExtraLifeTeamsUpdate, HandleExtraLifeTeamsUpdateTask)
	return mux
}
