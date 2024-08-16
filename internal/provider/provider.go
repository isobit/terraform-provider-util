// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the provider satisfies various provider interfaces.
var _ provider.Provider = &Provider{}
var _ provider.ProviderWithFunctions = &Provider{}

// Provider defines the provider implementation.
type Provider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type ProviderModel struct {
	BypassIndestructible types.Bool `tfsdk:"bypass_indestructible"`
}

type ProviderConfig struct {
	BypassIndestructible bool
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &Provider{
			version: version,
		}
	}
}

func (p *Provider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "util"
	resp.Version = p.version
}

func (p *Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"bypass_indestructible": schema.BoolAttribute{
				MarkdownDescription: "Globally bypasses destruction protection on util_indestructible resource that allow it.",
				Optional:            true,
			},
		},
	}
}

func (p *Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := ProviderConfig{
		BypassIndestructible: os.Getenv("TF_UTIL_BYPASS_INDESTRUCTIBLE") == "true",
	}
	if !data.BypassIndestructible.IsNull() {
		cfg.BypassIndestructible = data.BypassIndestructible.ValueBool()
	}
	resp.ResourceData = &cfg
}

func (p *Provider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewIndestructibleResource,
	}
}

func (p *Provider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *Provider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}
