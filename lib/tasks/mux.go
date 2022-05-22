package tasks

import "github.com/hibiken/asynq"

//GetMux maps names to handlers
func GetMux() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskExtraLifeTeamUpdate, HandleExtraLifeTeamUpdateTask)
	mux.HandleFunc(TaskExtraLifeParticipantUpdate, HandleExtraLifeParticipantUpdateTask)
	mux.HandleFunc(TaskExtraLifeTeamParticipantUpdate, HandleExtraLifeTeamUpdateParticipantTask)
	mux.HandleFunc(TaskExtraLifeTeamsUpdate, HandleExtraLifeTeamsUpdateTask)
	mux.HandleFunc(TaskExtraLifeParticipantsUpdate, HandleExtraLifeParticipantsUpdateTask)
	return mux
}
