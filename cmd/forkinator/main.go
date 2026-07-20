// SPDX-License-Identifier: GPL-3.0-or-later
// SPDX-FileCopyrightText: 2026 The Forkinator authors

package main

import (
	"fmt"
	"os"

	"github.com/jovens/forkinator/internal/git"
	"github.com/jovens/forkinator/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "forkinator",
	Short:         "GIT FORKINATOR",
	Long:          "Efficient Git repository forking using alternates. Create shared forks that save space by referencing the upstream repository's object store.",
	SilenceErrors: true,
	SilenceUsage:  true,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var forkCmd = &cobra.Command{
	Use:   "fork [upstream_path] [fork_path]",
	Short: "Create a new shared fork from an upstream repo",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		upstream := args[0]
		fork := args[1]

		err := git.CreateFork(upstream, fork)
		if err != nil {
			fmt.Println(ui.ErrorMsg("%v", err))
			os.Exit(1)
		}
		fmt.Println(ui.SuccessMsg("Successfully created fork at %s using alternates from %s", fork, upstream))
	},
}

var detachCmd = &cobra.Command{
	Use:   "detach [fork_path]",
	Short: "Make a fork independent of its upstream",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fork := args[0]

		fmt.Printf("• Detaching fork at %s (this may take a while)...\n", fork)
		err := git.DetachFork(fork)
		if err != nil {
			fmt.Println(ui.ErrorMsg("%v", err))
			os.Exit(1)
		}
		fmt.Println(ui.SuccessMsg("Successfully detached fork. It is now independent."))
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync [fork_path]",
	Short: "Sync a fork's references with its upstream repo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fork := args[0]

		fmt.Printf("• Syncing fork at %s with upstream...\n", fork)
		err := git.SyncFork(fork)
		if err != nil {
			fmt.Println(ui.ErrorMsg("%v", err))
			os.Exit(1)
		}
		fmt.Println(ui.SuccessMsg("Successfully synced fork references with upstream."))
	},
}

func init() {
	rootCmd.SetHelpFunc(ui.HandleHelp)
	rootCmd.SetUsageFunc(ui.HandleUsage)
	rootCmd.AddCommand(forkCmd)
	rootCmd.AddCommand(detachCmd)
	rootCmd.AddCommand(syncCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(ui.ErrorMsg("%v", err))
		os.Exit(1)
	}
}
