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
	"context"
	"crypto/tls"
	"github.com/fragforce/fragevents/lib/tasks"
	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"time"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Backend workers",
	Run: func(cmd *cobra.Command, args []string) {
		srv := asynq.NewServer(
			buildRedisConn(),
			asynq.Config{
				Concurrency: viper.GetInt("asynq.workers"),
			},
		)

		mux := asynq.NewServeMux()
		mux.HandleFunc(tasks.TaskExtraLifeUpdate, tasks.HandleExtraLifeUpdateTask)

		if err := srv.Run(mux); err != nil {
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
}

//parseRedisURL returns a parsed copy of the redis url. WARN: Does a FATAL on failure to parse.
func parseRedisURL() *url.URL {
	log := log
	rawURL := os.Getenv("REDIS_URL")
	log = log.WithField("url", rawURL)

	parsedRedisURL, err := url.Parse(rawURL)
	if err != nil {
		log.WithError(err).WithField("url", rawURL).Fatal("Failed to parse Redis URL")
	}

	_, ok := parsedRedisURL.User.Password()
	if !ok {
		log.Warn("No redis password")
	}

	return parsedRedisURL
}

func buildRedisConn() asynq.RedisClientOpt {
	parsedRedisURL := parseRedisURL()
	passwd, _ := parsedRedisURL.User.Password()

	return asynq.RedisClientOpt{
		Addr:         parsedRedisURL.Host,
		Password:     passwd,
		DB:           viper.GetInt("asynq.rdb"),
		DialTimeout:  viper.GetDuration("redis.dialtimeout"),
		ReadTimeout:  viper.GetDuration("redis.readtimeout"),
		WriteTimeout: viper.GetDuration("redis.writetimeout"),
		PoolSize:     viper.GetInt("redis.poolsize"),
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}

func getRedisClient() *redis.Client {
	parsedRedisURL := parseRedisURL()
	passwd, _ := parsedRedisURL.User.Password()

	rdb := redis.NewClient(&redis.Options{
		Addr:     parsedRedisURL.Host,
		Password: passwd,
		DB:       viper.GetInt("groupcache.rdb"),
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		panic("Problem connecting to redis: " + err.Error())
	}

	return rdb
}
