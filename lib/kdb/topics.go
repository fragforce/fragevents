package kdb

import (
	"fmt"
	"github.com/spf13/viper"
)

//MakeTopicName creates a topic name - adds the prefix if needed
func MakeTopicName(topicType string) string {
	prefix := viper.GetString("runtime.prefix")
	if prefix != "" {
		// Prefix will include a trailing dot, if it's not ""
		return fmt.Sprintf("%s%s", prefix, topicType)
	}
	return topicType
}
