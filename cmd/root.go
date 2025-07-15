package cmd

import (
	"log"
	"strings"

	"github.com/platform-mesh/virtual-workspaces/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	genericapiserveroptions "k8s.io/apiserver/pkg/server/options"
)

var (
	v                              *viper.Viper
	cfg                            config.ServiceConfig
	secureServing                  = genericapiserveroptions.SecureServingOptions{}
	delegatingAuthenticationOption = genericapiserveroptions.DelegatingAuthenticationOptions{}
)

var rootCmd = &cobra.Command{
	Use:   "virtual-workspaces",
	Short: "The Platform-Mesh virtual workspace",
}

func init() {
	v = viper.NewWithOptions(
		viper.EnvKeyReplacer(strings.NewReplacer("-", "_")),
	)

	v.AutomaticEnv()

	rootCmd.AddCommand(startCmd)

	err := config.BindConfigToFlags(v, startCmd, &cfg)
	if err != nil {
		log.Fatalln(err)
	}

	delegatingAuthenticationOption = *genericapiserveroptions.NewDelegatingAuthenticationOptions()

	delegatingAuthenticationOption.AddFlags(startCmd.Flags())
	secureServing.AddFlags(startCmd.Flags())
}

func Execute() { // coverage-ignore
	cobra.CheckErr(rootCmd.Execute())
}
