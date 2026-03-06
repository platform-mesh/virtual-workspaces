package cmd

import (
	"flag"

	"github.com/platform-mesh/virtual-workspaces/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	genericapiserveroptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/klog/v2"
)

var (
	cfg                            = config.NewServiceConfig()
	secureServing                  = genericapiserveroptions.SecureServingOptions{}
	delegatingAuthenticationOption = genericapiserveroptions.DelegatingAuthenticationOptions{}
)

var rootCmd = &cobra.Command{
	Use:   "virtual-workspaces",
	Short: "The Platform-Mesh virtual workspace",
}

func init() {
	rootCmd.AddCommand(startCmd)
	cfg.AddFlags(startCmd.Flags())

	delegatingAuthenticationOption = *genericapiserveroptions.NewDelegatingAuthenticationOptions()

	delegatingAuthenticationOption.AddFlags(startCmd.Flags())
	secureServing.AddFlags(startCmd.Flags())

	klogFlagSet := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlagSet)

	pflag.CommandLine.AddGoFlagSet(klogFlagSet)
	rootCmd.PersistentFlags().AddGoFlagSet(klogFlagSet)
}

func Execute() { // coverage-ignore
	defer klog.Flush()
	cobra.CheckErr(rootCmd.Execute())
}
