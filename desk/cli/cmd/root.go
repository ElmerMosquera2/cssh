/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cssh",
	Short: "Gestor de perfiles SSH y atadura de identidad por repositorio",
	Long: `CSSH gestiona perfiles SSH (clave + identidad Git + hosts)
y ata cada repositorio a un perfil en el momento del clone.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
