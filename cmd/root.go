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
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var cfgFile string
var rootLog *logrus.Logger
var log *logrus.Entry
var AmDebugging bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fragevents",
	Short: "A web app for donation events and info",
	Long:  `See fragforce.org & https://github.com/fragforce/fragevents `,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log := log.WithFields(logrus.Fields{
			"args": args,
		})
		if log != nil {
			log.Debug("Starting up")
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		log := log.WithFields(logrus.Fields{
			"args": args,
		})
		if log != nil {
			log.Debug("All done")
		}
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	//viper.SetDefault("log.level",logrus.DebugLevel)
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

	viper.SetEnvPrefix("CFG")
	viper.AutomaticEnv() // read in environment variables that match CFG_XXXXXXX

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
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

	if AmDebugging {
		rootLog.SetLevel(logrus.DebugLevel)
	}

	log.Info("Init'ed logging")
}
