package cmd

import (
	"flag"
	"log"
	"strings"

	"github.com/platform-mesh/virtual-workspaces/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	genericapiserveroptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/klog/v2"
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

	klogFlagSet := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlagSet)

	klogFlagSet.Set("logtostderr", "true")
	pflag.CommandLine.AddGoFlagSet(klogFlagSet)
	rootCmd.PersistentFlags().AddGoFlagSet(klogFlagSet)
}

func Execute() { // coverage-ignore
	defer klog.Flush()
	cobra.CheckErr(rootCmd.Execute())
}
