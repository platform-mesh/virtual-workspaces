package config

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ServiceConfig struct {
	ProviderWorkspaceID string `mapstructure:"provider-workspace-id" default:"2pkfvpweuy7symvj"`
	Kubeconfig          string `mapstructure:"kubeconfig"`
	ServerURL           string `mapstructure:"server-url"`
	EntityLabel         string `mapstructure:"entity-label" default:"ui.platform-mesh.ui/entity"`
	ContentForLabel     string `mapstructure:"content-for-label" default:"ui.platform-mesh.io/content-for"`

	ResourceSchemaName      string `mapstructure:"resource-schema-name" default:"v250704-6d57f16.contentconfigurations.core.openmfp.io"`
	ResourceSchemaWorkspace string `mapstructure:"resource-schema-workspace" default:"root:openmfp-system"`

	ProviderMetadataVirtualWorkspacePath string `mapstructure:"provider-metadata-virtual-workspace-path" default:"/services/apiexport/2pkfvpweuy7symvj/core.platform-mesh.io"`
	APIExportVirtualWorkspacePath        string `mapstructure:"apiexport-virtual-workspace-path" default:"/services/apiexport/2pkfvpweuy7symvj/core.platform-mesh.io"`
}

// generateFlagSet generates a pflag.FlagSet from a struct based on its `mapstructure` tags.
func generateFlagSet(config any) (*pflag.FlagSet, error) {
	flagSet := pflag.NewFlagSet("generated", pflag.ContinueOnError)
	err := traverseStruct(reflect.ValueOf(config), flagSet, "")
	return flagSet, err
}

// traverseStruct recursively traverses a struct and adds flags to the FlagSet.
func traverseStruct(value reflect.Value, flagSet *pflag.FlagSet, prefix string) error {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return errors.New("value must be a struct")
	}

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		// Get the `mapstructure` tag
		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}

		defaultValueTag := field.Tag.Get("default")
		defaultStrValue := ""
		if defaultValueTag != "" {
			defaultStrValue = defaultValueTag
		}

		descriptionValueTag := field.Tag.Get("description")
		descriptionStrValue := ""
		if descriptionValueTag != "" {
			descriptionStrValue = descriptionValueTag
		}

		// Handle nested structs
		if fieldValue.Kind() == reflect.Struct {
			if tag == ",squash" {
				err := traverseStruct(fieldValue, flagSet, "")
				if err != nil {
					return err
				}
			} else {
				err := traverseStruct(fieldValue, flagSet, prefix+tag+".")
				if err != nil {
					return err
				}
			}
			continue
		}

		description := fmt.Sprintf("Set the %s", tag)
		if descriptionStrValue != "" {
			description = descriptionStrValue
		}

		// Add flags based on the field type
		switch fieldValue.Kind() {
		case reflect.String:
			flagSet.String(prefix+tag, defaultStrValue, description)
		case reflect.Int, reflect.Int64:
			if fieldValue.Type() == reflect.TypeOf(time.Duration(0)) {
				var durVal time.Duration
				if defaultStrValue != "" {
					parsedDurVal, err := time.ParseDuration(defaultStrValue)
					if err != nil {
						return fmt.Errorf("invalid duration value for field %s: %w", field.Name, err)
					}
					durVal = parsedDurVal
				}

				durDescription := fmt.Sprintf("Set the %s in seconds", tag)
				if descriptionStrValue != "" {
					durDescription = descriptionStrValue
				}
				flagSet.Duration(prefix+tag, durVal, durDescription)
			} else {
				i := 0
				if defaultStrValue != "" {
					parsedInt, err := strconv.Atoi(defaultStrValue)
					if err != nil {
						return err
					}
					i = parsedInt
				}
				flagSet.Int(prefix+tag, i, description)
			}
		case reflect.Bool:
			var defaultBoolValue bool
			if defaultStrValue != "" {
				b, err := strconv.ParseBool(defaultStrValue)
				if err != nil {
					return err
				}
				defaultBoolValue = b
			}
			flagSet.Bool(prefix+tag, defaultBoolValue, description)
		default:
			return fmt.Errorf("unsupported field type %s for field %s", fieldValue.Kind(), field.Name)
		}

	}
	return nil
}

func BindConfigToFlags(v *viper.Viper, cmd *cobra.Command, config any) error {
	flagSet, err := generateFlagSet(config)
	if err != nil {
		return fmt.Errorf("failed to generate flag set: %w", err)
	}
	err = v.BindPFlags(flagSet)
	if err != nil {
		return err
	}

	cmd.Flags().AddFlagSet(flagSet)

	cobra.OnInitialize(unmarshalIntoStruct(v, config))

	return nil
}

func unmarshalIntoStruct(v *viper.Viper, cfg any) func() {
	return func() {
		if err := v.Unmarshal(cfg); err != nil {
			panic(fmt.Errorf("failed to unmarshal config: %w", err))
		}
	}
}
