package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	name := path.Base(os.Args[0])

	// set the umask to 022 so that we can create files/directories with 755 permissions
	// this does not return an error - it returns the previous umask
	// we do this before calling cli.InitAndExecute so that it is set before the process forks
	_ = syscall.Umask(0o022)

	InitAndExecute(ctx, name)
}

func RootCmd(ctx context.Context, v *viper.Viper, name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "Run or pull data for the local artifact mirror",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			v.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			os.Exit(1)
			return nil
		},
	}

	cobra.OnInitialize(func() {
		initConfig(v)
	})

	cmd.AddCommand(ServeCmd(ctx, v))
	cmd.AddCommand(PullCmd(ctx, v))

	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return cmd
}

func InitAndExecute(ctx context.Context, name string) {
	v := viper.GetViper()
	if err := RootCmd(ctx, v, name).Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig(v *viper.Viper) {
	v.SetEnvPrefix("REPLICATED")
	v.AutomaticEnv()
}
