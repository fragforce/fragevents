package kdb

import "github.com/spf13/viper"

func init() {
	viper.SetDefault("kafka.topics.events", "events")
	viper.SetDefault("kafka.topics.teams", "teams")
	viper.SetDefault("kafka.topics.participants", "participants")
	viper.SetDefault("kafka.topics.donations", "donations")
}
