/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// profileCmd represents the profile command
// var profileCmd = &cobra.Command{
// 	Use:   "profile",
// 	Short: "A brief description of your command",
// 	Long: `A longer description that spans multiple lines and likely contains examples
// and usage of using your command. For example:

// Cobra is a CLI library for Go that empowers applications.
// This application is a tool to generate the needed files
// to quickly create a Cobra application.`,
// 	Run: func(cmd *cobra.Command, args []string) {
// 		fmt.Println("profile called")
// 	},
// }

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Gestiona perfiles SSH",
}

func init() {
	rootCmd.AddCommand(profileCmd)
}
