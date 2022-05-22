package cmd

/*
Copyright © 2022 Paulson McIntyre <paulson@fragforce.org>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

import (
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/tasks"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Backend workers",
	Run: func(cmd *cobra.Command, args []string) {
		srv := asynq.NewServer(
			df.BuildAsyncQRedis(),
			asynq.Config{
				Concurrency: viper.GetInt("asynq.workers"),
			},
		)

		go func() {
			if err := ginEngine.Run(viper.GetString("listen") + ":" + viper.GetString("port")); err != nil {
				log.WithError(err).Fatal("Problem running GIN")
			}
		}()

		if err := srv.Run(tasks.GetMux()); err != nil {
			log.WithError(err).Fatal("Problem running asynq worker daemon")
		}
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
	//workerCmd.PersistentFlags().String("queue", "", "A help for foo")
	viper.SetDefault("asynq.rdb", 1)
	viper.SetDefault("redis.dialtimeout", time.Second*5)
	viper.SetDefault("redis.readtimeout", time.Second*15)
	viper.SetDefault("redis.writetimeout", time.Second*15)
	viper.SetDefault("redis.poolsize", 120)
	viper.SetDefault("asynq.workers", 32)
	viper.SetDefault("groupcache.rdb", 2)
	viper.SetDefault("groupcache.redis.retries", 6)
}
