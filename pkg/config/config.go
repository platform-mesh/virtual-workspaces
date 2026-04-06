package config

import "github.com/spf13/pflag"

type ServiceConfig struct {
	Kubeconfig        string
	ServerURL         string
	EntityLabel       string
	ContentForLabel   string
	MainEntityName    string
	AccountEntityName     string
	MarketplaceScopeLabel string

	ResourceSchemaName      string
	ResourceSchemaWorkspace string

	ResourceAPIExportEndpointSliceName string
}

func NewServiceConfig() ServiceConfig {
	return ServiceConfig{
		EntityLabel:             "ui.platform-mesh.ui/entity",
		ContentForLabel:         "ui.platform-mesh.io/content-for",
		MainEntityName:          "main",
		AccountEntityName:       "core_platform-mesh_io_account",
		MarketplaceScopeLabel:   "ui.platform-mesh.io/scope",
		ResourceSchemaName:      "v250704-6d57f16.contentconfigurations.ui.platform-mesh.io",
		ResourceSchemaWorkspace: "root:openmfp-system",
	}
}

func (c *ServiceConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Kubeconfig, "kubeconfig", c.Kubeconfig, "Set the kubeconfig file path")
	fs.StringVar(&c.ServerURL, "server-url", c.ServerURL, "Set the server URL")
	fs.StringVar(&c.EntityLabel, "entity-label", c.EntityLabel, "Set the entity label")
	fs.StringVar(&c.ContentForLabel, "content-for-label", c.ContentForLabel, "Set the content-for label")
	fs.StringVar(&c.MainEntityName, "main-entity-name", c.MainEntityName, "Set the main entity name")
	fs.StringVar(&c.AccountEntityName, "account-entity-name", c.AccountEntityName, "Set the account entity name")
	fs.StringVar(&c.MarketplaceScopeLabel, "marketplace-scope-label", c.MarketplaceScopeLabel, "Label used to scope providers to org or account level")
	fs.StringVar(&c.ResourceSchemaName, "resource-schema-name", c.ResourceSchemaName, "Set the resource schema name")
	fs.StringVar(&c.ResourceSchemaWorkspace, "resource-schema-workspace", c.ResourceSchemaWorkspace, "Set the resource schema workspace")
	fs.StringVar(
		&c.ResourceAPIExportEndpointSliceName,
		"resource-apiexport-endpointslice-name",
		c.ResourceAPIExportEndpointSliceName,
		"Set the resource APIExport EndpointSlice name",
	)
}
