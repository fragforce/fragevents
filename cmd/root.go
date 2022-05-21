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
	"fmt"
	"github.com/fragforce/fragevents/lib/df"
	"github.com/fragforce/fragevents/lib/gcache"
	"github.com/fragforce/fragevents/lib/kdb"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"strings"
)

var cfgFile string
var rootLog *logrus.Logger
var log *logrus.Entry
var AmDebugging bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fragevents",
	Short: "A web app for donation events and info",
	Long:  `See fragforce.org & https://github.com/fragforce/fragevents`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log := log.WithFields(logrus.Fields{
			"args": args,
		})

		log.Info("Setting up gcache")
		gca, err := gcache.NewGlobalSharedGCache(log, viper.GetString("groupcache.basedir"), getRedisClient())
		if err != nil {
			log.WithError(err).Fatal("Problem setting up global shared groupcache")
			return
		}
		if err := gca.StartRun(nil); err != nil {
			log.WithError(err).Fatal("Problem starting up global shared groupcache")
			return
		}

		log.Debug("Starting up")
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		log := log.WithFields(logrus.Fields{
			"args": args,
		})
		gca := gcache.GlobalCache()

		// Flush dem buffers
		log.Info("Closing out kafka writers")
		if err := kdb.W.Close(); err != nil {
			log.WithError(err).Error("Problem closing out all kafka writers")
		}

		if err := gca.Shutdown(); err != nil {
			log.WithError(err).Error("Problem shutting down groupcache server")
		}

		log.Debug("All done")
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	viper.SetDefault("log.level", logrus.DebugLevel)
	viper.SetDefault("runtime.app_id", "x")
	viper.SetDefault("runtime.app_name", "fragevents")
	viper.SetDefault("runtime.dyno_id", "x")
	viper.SetDefault("runtime.release_created_at", "x")
	viper.SetDefault("runtime.release_version", "v0")
	viper.SetDefault("runtime.slug_commit", "00")
	viper.SetDefault("runtime.slug_description", "Blah")

	viper.SetDefault("kafka.urls", []string{})

	cobra.OnInitialize(initConfig, initLogging)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fragevents.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&AmDebugging, "debug", "d", false, "Enable debug mode")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".fragevents" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".fragevents")
	}

	replacer := strings.NewReplacer("-", "_", ".", "_", " ", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.SetEnvPrefix("CFG")
	viper.AutomaticEnv() // read in environment variables that match CFG_XXXXXXX

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if _, e2 := fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed()); e2 != nil {
			panic("Error writing to stderr: " + e2.Error())
		}
	}

	kURL := os.Getenv("KAFKA_URL")
	if kURL != "" {
		var urls []string
		for _, u := range strings.Split(kURL, ",") {
			parsed, err := url.Parse(u)
			if err != nil {
				panic("Bad URL in kafka urls: " + err.Error() + " : " + u)
			}
			urls = append(urls, parsed.String())
		}
		viper.Set("kafka.urls", urls)
	}

	// Set heroku env var and such
	for _, e := range []string{
		// https://devcenter.heroku.com/articles/dyno-metadata#dyno-metadata
		"HEROKU_APP_ID",
		"HEROKU_APP_NAME",
		"HEROKU_DYNO_ID",
		"HEROKU_RELEASE_CREATED_AT",
		"HEROKU_RELEASE_VERSION",
		"HEROKU_SLUG_COMMIT",
		"HEROKU_SLUG_DESCRIPTION",
		"KAFKA_TRUSTED_CERT",
		"KAFKA_CLIENT_CERT",
		"KAFKA_CLIENT_CERT_KEY",
		"KAFKA_URL",
		"KAFKA_PREFIX",
	} {
		k := fmt.Sprintf("runtime.%v", strings.ReplaceAll(strings.ToLower(e), "heroku_", ""))
		v := os.Getenv(e)
		if v != "" {
			viper.Set(k, v)
		}
	}

	// Update debug mode from config
	if !AmDebugging && viper.GetBool("debug") {
		AmDebugging = true
	}

	// Save PORT env
	if p := os.Getenv("PORT"); p != "" {
		viper.Set("port", p)
	}
}

func initLogging() {
	rootLog = logrus.New()
	lvl, err := logrus.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		panic("Bad log level: " + err.Error())
	}
	rootLog.SetLevel(lvl)
	rootLog.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp:  false,
		DisableHTMLEscape: false,
	})
	rootLog.SetReportCaller(true)
	log = rootLog.WithFields(logrus.Fields{
		"app": rootCmd.Name(),
	})

	if AmDebugging && lvl > logrus.DebugLevel {
		rootLog.SetLevel(logrus.DebugLevel)
	}

	for k, v := range viper.GetStringMapString("runtime") {
		if strings.Contains(strings.ToLower(k), "cert") || strings.Contains(strings.ToLower(v), "certificate") {
			// has cert data
			continue
		}
		log = log.WithField(k, v)
	}

	// Set global logger
	df.Log = log

	log.Info("Init'ed logging")
}
