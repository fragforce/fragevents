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
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Web frontend worker",
	Run: func(cmd *cobra.Command, args []string) {
		r := gin.Default()
		r.GET("/alive", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"alive": true,
				"ok":    true,
				"error": nil,
			})
		})
		if err := r.Run(); err != nil {
			log.WithError(err).Fatal("Problem running GIN")
		}
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
}
