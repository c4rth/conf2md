package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"

	"conf2md/internal/config"
	"conf2md/internal/converter"
	"conf2md/internal/goconfluence"
)

var version = "dev"

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := newRootCommand().ExecuteContext(ctx)
	stop()
	if err != nil {
		log.Error().Err(err).Msg("conf2md encountered an error")
		os.Exit(1)
	}
}

func initConfig(settings *viper.Viper) error {
	if err := gotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Warn().Err(err).Msg("continuing without .env file")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	settings.AddConfigPath(cwd)
	settings.SetConfigName("config")
	settings.SetEnvPrefix("CONF2MD")
	settings.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	settings.AutomaticEnv()

	if err := settings.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return fmt.Errorf("read configuration: %w", err)
		}
	} else {
		log.Info().Msgf("using config file: %s", settings.ConfigFileUsed())
	}
	return nil
}

func resolveArgs(settings *viper.Viper) config.CmdLineArgs {
	return config.CmdLineArgs{
		Space:          settings.GetString("space"),
		Page:           settings.GetString("page"),
		OutputDir:      settings.GetString("output"),
		ConfluenceUrl:  settings.GetString("url"),
		ConfluenceApi:  settings.GetString("api"),
		Token:          settings.GetString("token"),
		ExportChildren: settings.GetBool("children"),
		Local:          settings.GetBool("local"),
		Mkdocs:         settings.GetBool("mkdocs"),
		Debug:          settings.GetBool("debug"),
		Timeout:        settings.GetDuration("timeout"),
	}
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
			if err := initConfig(settings); err != nil {
				return err
			}
			args = resolveArgs(settings)
			if args.Debug {
				log.Logger = log.Logger.Level(zerolog.DebugLevel)
			} else {
				log.Logger = log.Logger.Level(zerolog.InfoLevel)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := args.Validate(); err != nil {
				_ = cmd.Help()
				return err
			}
			return converter.ConvertConfluenceToMarkdown(cmd.Context(), &args)
		},
	}
	command.Version = version
	command.Flags().StringVarP(&args.Space, "space", "s", "", "Confluence space key (env: CONF2MD_SPACE)")
	command.Flags().StringVarP(&args.Page, "page", "p", "", "Confluence 'pageId', 'space/title' or file name with path (env: CONF2MD_PAGE)")
	command.Flags().StringVarP(&args.OutputDir, "output", "o", "output", "Directory to save Markdown files (default: output, env: CONF2MD_OUTPUT)")
	command.Flags().StringVarP(&args.ConfluenceUrl, "url", "u", "", "Confluence base URL (env: CONF2MD_URL)")
	command.Flags().StringVarP(&args.ConfluenceApi, "api", "a", "/wiki/rest/api", "Confluence API endpoint (default: /wiki/rest/api, env: CONF2MD_API)")
	command.Flags().StringVarP(&args.Token, "token", "t", "", "Confluence API token (env: CONF2MD_TOKEN)")
	command.Flags().BoolVarP(&args.ExportChildren, "children", "c", true, "export child pages (default: true, env: CONF2MD_CHILDREN)")
	command.Flags().BoolVarP(&args.Local, "local", "l", false, "use local mode (env: CONF2MD_LOCAL)")
	command.Flags().BoolVarP(&args.Mkdocs, "mkdocs", "m", true, "generate mkdocs.yml (default: true, env: CONF2MD_MKDOCS)")
	command.Flags().BoolVarP(&args.Debug, "debug", "d", false, "enable debug logging (env: CONF2MD_DEBUG)")
	command.Flags().DurationVar(&args.Timeout, "timeout", goconfluence.DefaultTimeout, "maximum duration of each Confluence HTTP request, 0 disables it (default: "+goconfluence.DefaultTimeout.String()+", env: CONF2MD_TIMEOUT)")
	cobra.CheckErr(settings.BindPFlags(command.Flags()))
	return command
}
