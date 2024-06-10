package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IndestructibleResource{}

type IndestructibleResourceModel struct {
	AllowDestroy types.Bool   `tfsdk:"allow_destroy"`
	ErrorMessage types.String `tfsdk:"error_message"`
}

type IndestructibleResource struct{}

func NewIndestructibleResource() resource.Resource {
	return &IndestructibleResource{}
}

func (r *IndestructibleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_indestructible"
}

func (r *IndestructibleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
The ` + "`util_indestructible`" + ` resource creates a node in the resource
graph that cannot be destroyed unless the ` + "`allow_destroy`" + ` attribute
is set to ` + "`true`" + ` in the state. This provides a workaround to a
[well-known shortcoming](https://github.com/hashicorp/terraform/issues/17599)
of the ` + "`prevent_destroy`" + ` lifecycle attribute by exploiting
dependency order to prevent the destruction of resources that are dependencies
of the indestructible resource, since it must be destroyed before the dependencies
are.
`,

		Attributes: map[string]schema.Attribute{
			"allow_destroy": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow destruction.",
				Optional:            true,
			},
			"error_message": schema.StringAttribute{
				MarkdownDescription: "Additional message to include in the error message when attempting to destroy when `allow_destroy` is `false`.",
				Optional:            true,
			},
		},
	}
}

func (r *IndestructibleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IndestructibleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IndestructibleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// no-op
}

func (r *IndestructibleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IndestructibleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IndestructibleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IndestructibleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.AllowDestroy.ValueBool() {
		msg := "Cannot destroy indestructible resource unless allow_destroy is set in state. To continue with destruction, set allow_destroy to true and apply, and then remove."
		if extraMsg := data.ErrorMessage.ValueString(); extraMsg != "" {
			msg += "\n\n" + extraMsg
		}
		resp.Diagnostics.AddError("Not Allowed", msg)
		return
	}
}
