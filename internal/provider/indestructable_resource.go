// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	// "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	// "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	// "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IndestructableResource{}

func NewIndestructableResource() resource.Resource {
	return &IndestructableResource{}
}

// IndestructableResource defines the resource implementation.
type IndestructableResource struct {}

// IndestructableResourceModel describes the resource data model.
type IndestructableResourceModel struct {
	AllowDestroy types.Bool `tfsdk:"allow_destroy"`
}

func (r *IndestructableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_indestructable"
}

func (r *IndestructableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Indestructable resource",

		Attributes: map[string]schema.Attribute{
			"allow_destroy": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow destruction",
				Optional:            true,
			},
		},
	}
}

func (r *IndestructableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IndestructableResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IndestructableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// var data IndestructableResourceModel

	// // Read Terraform prior state data into the model
	// resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// if resp.Diagnostics.HasError() {
	// 	return
	// }

	// // Save updated data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IndestructableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IndestructableResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IndestructableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IndestructableResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.AllowDestroy.ValueBool() {
		resp.Diagnostics.AddError("Not Allowed", "cannot destroy indestructable resource unless allow_destroy is set in state")
		return
	}
}
