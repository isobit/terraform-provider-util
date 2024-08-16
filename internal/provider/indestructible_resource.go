package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IndestructibleResource{}

type IndestructibleResourceModel struct {
	AllowDestroy   types.Bool    `tfsdk:"allow_destroy"`
	AllowBypass    types.Bool    `tfsdk:"allow_bypass"`
	ErrorMessage   types.String  `tfsdk:"error_message"`
	ProtectedValue types.Dynamic `tfsdk:"protected_value"`
}

type IndestructibleResource struct {
	Bypass bool
}

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
			"allow_bypass": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow destruction when `bypass_indestructible` is set on the provider.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"error_message": schema.StringAttribute{
				MarkdownDescription: "Additional message to include in the error message when attempting to destroy when `allow_destroy` is `false`.",
				Optional:            true,
			},
			"protected_value": schema.DynamicAttribute{
				MarkdownDescription: "",
				Optional:            true,
				PlanModifiers: []planmodifier.Dynamic{
					dynamicplanmodifier.RequiresReplaceIf(
						func(ctx context.Context, req planmodifier.DynamicRequest, resp *dynamicplanmodifier.RequiresReplaceIfFuncResponse) {
							if req.StateValue.IsNull() {
								return
							}
							resp.RequiresReplace = true
						},
						"",
						"",
					),
				},
			},
		},
	}
}

func (r *IndestructibleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	cfg, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.Bypass = cfg.BypassIndestructible
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

	if data.AllowDestroy.ValueBool() {
		return
	}

	bypass := r.Bypass && data.AllowBypass.ValueBool()
	if bypass {
		resp.Diagnostics.AddWarning("Bypassing Destroy Protection", "Proceeding to destroy util_indestructible instance due to bypass_indestructible being true on the provider, and allow_bypass being true on the resource.")
		return
	}

	msg := "Cannot destroy indestructible resource unless allow_destroy is set in state.\n" +
		"To continue with destruction, set allow_destroy to true and apply first.\n" +
		"Or, unless otherwise prohibited, set bypass_indestructible on the provider, or set\n" +
		"the TF_UTIL_BYPASS_INDESTRUCTIBLE env var to \"true\"."
	if extraMsg := data.ErrorMessage.ValueString(); extraMsg != "" {
		msg += "\n\n" + extraMsg
	}
	resp.Diagnostics.AddError("Destruction Not Allowed", msg)
}
