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
	"github.com/fragforce/fragevents/lib/tasks"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)

// schedCmd represents the sched command
var schedCmd = &cobra.Command{
	Use:   "sched",
	Short: "A backend worker that triggers timed tasks",
	Run: func(cmd *cobra.Command, args []string) {
		loc, err := time.LoadLocation("UTC")
		if err != nil {
			panic(err)
		}
		scheduler := asynq.NewScheduler(
			buildRedisConn(),
			&asynq.SchedulerOpts{
				Location: loc,
			},
		)

		// Register cron tasks
		tasks.RegisterSched(scheduler)

		go func() {
			if err := ginEngine.Run(viper.GetString("listen") + ":" + viper.GetString("port")); err != nil {
				log.WithError(err).Fatal("Problem running GIN")
			}
		}()

		if err := scheduler.Run(); err != nil {
			log.WithError(err).Fatal("Problem running asynq scheduler daemon")
		}
	},
}

func init() {
	rootCmd.AddCommand(schedCmd)
}
