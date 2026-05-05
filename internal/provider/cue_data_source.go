package provider

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CueDataSource{}

type CueDataSource struct{}

type CueDataSourceModel struct {
	Path    types.String  `tfsdk:"path"`
	Package types.String  `tfsdk:"package"`
	Tags    types.Map     `tfsdk:"tags"`
	Input   types.List    `tfsdk:"input"`
	Result  types.Dynamic `tfsdk:"result"`
}

func NewCueDataSource() datasource.DataSource {
	return &CueDataSource{}
}

func (d *CueDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cue"
}

func (d *CueDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Evaluates a [CUE](https://cuelang.org/) file or package and exposes the result as a dynamic Terraform value.",

		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "Path to a CUE file or directory to evaluate. When a directory is specified, all `.cue` files in that directory sharing the same package clause are loaded as a package. Relative paths are resolved from the Terraform working directory.",
				Required:            true,
			},
			"package": schema.StringAttribute{
				MarkdownDescription: "Package name to select when loading a directory that contains multiple packages. Ignored for single-file loading.",
				Optional:            true,
			},
			"tags": schema.MapAttribute{
				MarkdownDescription: "Tags to inject into the CUE evaluation (equivalent to `cue eval -t key=value`). Values are strings; use `@tag(name, type=int)` in CUE for typed injection.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"input": schema.ListAttribute{
				MarkdownDescription: "Additional CUE expressions to unify with the loaded file. Useful for applying inline constraints or merging additional configuration.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"result": schema.DynamicAttribute{
				MarkdownDescription: "The evaluated CUE value as a native Terraform type.",
				Computed:            true,
			},
		},
	}
}

func (d *CueDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CueDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := data.Path.ValueString()

	// Build tags from the map.
	var tags []string
	if !data.Tags.IsNull() {
		elements := data.Tags.Elements()
		for k, v := range elements {
			tags = append(tags, k+"="+v.(types.String).ValueString())
		}
		sort.Strings(tags)
	}

	// Determine if path is a file or directory.
	info, err := os.Stat(path)
	if err != nil {
		resp.Diagnostics.AddError("CUE load error", fmt.Sprintf("Cannot stat path: %s", err))
		return
	}

	// Build load config.
	loadCfg := &load.Config{
		Tags: tags,
	}

	var loadArg string
	if info.IsDir() {
		// Directory: load as a package. The loader auto-detects cue.mod/ by
		// walking up from Dir.
		loadCfg.Dir = path
		loadArg = "."
		if !data.Package.IsNull() && data.Package.ValueString() != "" {
			loadCfg.Package = data.Package.ValueString()
		}
	} else {
		// Single file.
		loadArg = path
	}

	// Load and evaluate.
	cueCtx := cuecontext.New()
	insts := load.Instances([]string{loadArg}, loadCfg)
	if len(insts) == 0 {
		resp.Diagnostics.AddError("CUE load error", "No instances returned")
		return
	}
	inst := insts[0]
	if inst.Err != nil {
		resp.Diagnostics.AddError("CUE load error", formatCueError(inst.Err))
		return
	}

	val := cueCtx.BuildInstance(inst)
	if err := val.Err(); err != nil {
		resp.Diagnostics.AddError("CUE build error", formatCueError(err))
		return
	}

	// Unify with additional input if provided.
	if !data.Input.IsNull() {
		for i, elem := range data.Input.Elements() {
			s := elem.(types.String).ValueString()
			other := cueCtx.CompileString(s)
			if err := other.Err(); err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("CUE compilation error on input expression %d", i+1),
					formatCueError(err),
				)
				return
			}
			val = val.Unify(other)
		}
		if err := val.Err(); err != nil {
			resp.Diagnostics.AddError("CUE unification error", formatCueError(err))
			return
		}
	}

	if err := val.Validate(cue.Concrete(true)); err != nil {
		resp.Diagnostics.AddError("CUE validation error", formatCueError(err))
		return
	}

	// Convert CUE value to Terraform dynamic type.
	tfVal, diags := cueValueToTerraform(val)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Result = types.DynamicValue(tfVal)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// formatCueError formats a CUE error with source positions (file:line:col).
func formatCueError(err error) string {
	return strings.TrimSpace(errors.Details(err, nil))
}

// cueValueToTerraform converts a concrete cue.Value to a Terraform attr.Value.
func cueValueToTerraform(v cue.Value) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch v.Kind() {
	case cue.NullKind:
		return types.DynamicNull(), nil

	case cue.BoolKind:
		b, err := v.Bool()
		if err != nil {
			diags.AddError("CUE conversion error", fmt.Sprintf("Failed to get bool: %s", err))
			return nil, diags
		}
		return types.BoolValue(b), nil

	case cue.IntKind:
		var i big.Int
		n, err := v.Int(&i)
		if err != nil {
			diags.AddError("CUE conversion error", fmt.Sprintf("Failed to get int: %s", err))
			return nil, diags
		}
		_ = n
		return types.NumberValue(new(big.Float).SetInt(&i)), nil

	case cue.FloatKind:
		f, err := v.Float64()
		if err != nil {
			diags.AddError("CUE conversion error", fmt.Sprintf("Failed to get float: %s", err))
			return nil, diags
		}
		return types.NumberValue(new(big.Float).SetFloat64(f)), nil

	case cue.StringKind:
		s, err := v.String()
		if err != nil {
			diags.AddError("CUE conversion error", fmt.Sprintf("Failed to get string: %s", err))
			return nil, diags
		}
		return types.StringValue(s), nil

	case cue.ListKind:
		iter, err := v.List()
		if err != nil {
			diags.AddError("CUE conversion error", fmt.Sprintf("Failed to iterate list: %s", err))
			return nil, diags
		}

		var elems []attr.Value
		var elemTypes []attr.Type
		for iter.Next() {
			elem, d := cueValueToTerraform(iter.Value())
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			elems = append(elems, elem)
			elemTypes = append(elemTypes, elem.Type(context.Background()))
		}

		if len(elems) == 0 {
			return types.TupleValueMust([]attr.Type{}, []attr.Value{}), nil
		}

		return types.TupleValueMust(elemTypes, elems), nil

	case cue.StructKind:
		iter, err := v.Fields()
		if err != nil {
			diags.AddError("CUE conversion error", fmt.Sprintf("Failed to iterate struct: %s", err))
			return nil, diags
		}

		attrTypes := map[string]attr.Type{}
		attrValues := map[string]attr.Value{}
		for iter.Next() {
			label := iter.Selector().String()
			fieldVal, d := cueValueToTerraform(iter.Value())
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
			attrTypes[label] = fieldVal.Type(context.Background())
			attrValues[label] = fieldVal
		}

		objVal, objDiags := types.ObjectValue(attrTypes, attrValues)
		diags.Append(objDiags...)
		return objVal, diags

	default:
		diags.AddError("CUE conversion error", fmt.Sprintf("Unsupported CUE kind: %s", v.Kind()))
		return nil, diags
	}
}
