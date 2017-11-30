// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/nikogura/gomason/mason"
	"github.com/spf13/cobra"
	"log"
)

// signCmd represents the sign command
var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign your binaries after building them.",
	Long: `
Sign your binaries after building them.

Artists sign their work, you should too.

Signing sorta implies something to sign, which in turn, implies that it built, which means it tested successfully.  What I'm getting at is this command will run 'test', 'build', and then it will 'sign'.
`,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := mason.WholeShebang(workdir, branch, true, true, false, verbose)
		if err != nil {
			log.Fatalf("Error running sign: %s", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(signCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// signCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// signCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}