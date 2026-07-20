// SPDX-License-Identifier: GPL-3.0-or-later
// SPDX-FileCopyrightText: 2026 The Forkinator authors

package ui

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Logo = `
    /$$$$$$$$                  /$$       /$$                       /$$
   | $$_____/                 | $$      |__/                      | $$
   | $$     /$$$$$$   /$$$$$$ | $$   /$$ /$$ /$$$$$$$   /$$$$$$  /$$$$$$    /$$$$$$   /$$$$$$
   | $$$$$ /$$__  $$ /$$__  $$| $$  /$$/| $$| $$__  $$ |____  $$|_  $$_/   /$$__  $$ /$$__  $$
   | $$__/| $$  \ $$| $$  \__/| $$$$$$/ | $$| $$  \ $$  /$$$$$$$  | $$    | $$  \ $$| $$  \__/
   | $$   | $$  | $$| $$      | $$_  $$ | $$| $$  | $$ /$$__  $$  | $$ /$$| $$  | $$| $$
   | $$   |  $$$$$$/| $$      | $$ \  $$| $$| $$  | $$|  $$$$$$$  |  $$$$/|  $$$$$$/| $$
   |__/    \______/ |__/      |__/  \__/|__/|__/  |__/ \_______/   \___/   \______/ |__/
`

func HandleHelp(cmd *cobra.Command, args []string) {
	_ = cmd.Usage()
}

func HandleUsage(cmd *cobra.Command) error {
	fmt.Println(LogoStyle.Render(Logo))
	fmt.Println(HelpTitleStyle.Render(cmd.Short))
	fmt.Println(HelpDescStyle.Render(cmd.Long))

	fmt.Println(HelpSectionStyle.Render("USAGE"))
	if cmd.HasSubCommands() {
		fmt.Printf("  %s [command]\n", cmd.CommandPath())
	} else {
		fmt.Printf("  %s\n", cmd.UseLine())
	}

	if len(cmd.Commands()) > 0 {
		fmt.Println(HelpSectionStyle.Render("AVAILABLE COMMANDS"))
		for _, c := range cmd.Commands() {
			if !c.Hidden {
				fmt.Printf("  %-15s %s\n", c.Name(), c.Short)
			}
		}
	}

	if cmd.Flags().HasFlags() {
		fmt.Println(HelpSectionStyle.Render("FLAGS"))
		fmt.Println(HelpFlagStyle.Render(cmd.Flags().FlagUsages()))
	}

	if len(cmd.Commands()) > 0 {
		fmt.Println(HelpSectionStyle.Render("LEARN MORE"))
		fmt.Printf("  Use \"%s [command] --help\" for more information about a command.\n", cmd.CommandPath())
	}
	return nil
}
