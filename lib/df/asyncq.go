package df

import "github.com/hibiken/asynq"

var aClient *asynq.Client

//CreateAsyncQClient creates a brand-new client - used GetAsyncQClient normally
func CreateAsyncQClient() *asynq.Client {
	return asynq.NewClient(BuildAsyncQRedis())
}

func CreateSetCreateAsyncQClient() {
	aClient = CreateAsyncQClient()
}

//GetAsyncQClient returns an asyncq client
func GetAsyncQClient() *asynq.Client {
	if aClient == nil {
		CreateSetCreateAsyncQClient()
	}
	return aClient
}
