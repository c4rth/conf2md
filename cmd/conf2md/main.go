package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"

	"conf2md/internal/config"
	"conf2md/internal/converter"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if err := newRootCommand().Execute(); err != nil {
		log.Error().Err(err).Msg("conf2md encountered an error")
		os.Exit(1)
	}
}

func initConfig(settings *viper.Viper) error {
	if err := gotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msg("load .env")
		log.Warn().Msg("continuing without .env file")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	settings.AddConfigPath(cwd)
	settings.SetEnvPrefix("CONFLUENCE")
	settings.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	settings.AutomaticEnv()

	if err := settings.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("read configuration: %w", err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Using config file:", settings.ConfigFileUsed())
	}
	return nil
}

func newRootCommand() *cobra.Command {
	var args config.CmdLineArgs
	settings := viper.New()

	command := &cobra.Command{
		Use:           "conf2md",
		Short:         "Convert Confluence content to Markdown",
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if args.Debug {
				log.Logger = log.Logger.Level(zerolog.DebugLevel)
			} else {
				log.Logger = log.Logger.Level(zerolog.InfoLevel)
			}
			if err := initConfig(settings); err != nil {
				return err
			}
			args.ConfluenceUrl = settings.GetString("url")
			args.Token = settings.GetString("token")
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := args.Validate(); err != nil {
				return err
			}
			return converter.ConvertConfluenceToMarkdown(&args)
		},
	}
	command.Version = "0.1.0"
	command.Flags().StringVarP(&args.Page, "page", "p", "", "Confluence 'pageId', 'space/title' or file name with path")
	command.Flags().StringVarP(&args.OutputDir, "output", "o", "output", "Directory to save Markdown files")
	command.Flags().StringVarP(&args.ConfluenceUrl, "url", "u", "", "Confluence base URL")
	cobra.CheckErr(settings.BindPFlag("url", command.Flags().Lookup("url")))
	command.Flags().StringVarP(&args.Token, "token", "t", "", "Confluence API token")
	cobra.CheckErr(settings.BindPFlag("token", command.Flags().Lookup("token")))
	command.Flags().BoolVarP(&args.ExportChildren, "children", "c", true, "export child pages")
	command.Flags().BoolVarP(&args.Local, "local", "l", false, "use local mode")
	command.Flags().BoolVarP(&args.Mkdocs, "mkdocs", "m", true, "generate MkDocs compatible output")
	command.Flags().BoolVarP(&args.Debug, "debug", "d", false, "enable debug logging")
	return command
}
