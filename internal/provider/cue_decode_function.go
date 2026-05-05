package provider

import (
	"context"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = &CueDecodeFunction{}

type CueDecodeFunction struct{}

func NewCueDecodeFunction() function.Function {
	return &CueDecodeFunction{}
}

func (f *CueDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cuedecode"
}

func (f *CueDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Decodes one or more CUE strings into a dynamic Terraform value",
		MarkdownDescription: "Parses and evaluates one or more [CUE](https://cuelang.org/) expression strings, unifying them together " +
			"and returning the result as a native Terraform value. When multiple arguments are provided, " +
			"they are unified (merged with constraint checking) in order. This allows applying inline " +
			"constraints or overrides to loaded CUE configurations.\n\n" +
			"```hcl\n" +
			"# Simple decode\n" +
			"provider::util::cuedecode(file(\"config.cue\"))\n\n" +
			"# With inline constraints\n" +
			"provider::util::cuedecode(file(\"config.cue\"), <<CUE\n" +
			"  port: >1024\n" +
			"  name: =~\"^[a-z]+$\"\n" +
			"CUE\n)\n" +
			"```",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "cue",
				MarkdownDescription: "The primary CUE expression string to evaluate.",
			},
		},
		VariadicParameter: function.StringParameter{
			Name:                "constraints",
			MarkdownDescription: "Additional CUE expressions to unify with the primary value. Useful for applying constraints or merging configurations.",
		},
		Return: function.DynamicReturn{},
	}
}

func (f *CueDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var extra []string
	resp.Error = req.Arguments.Get(ctx, &input, &extra)
	if resp.Error != nil {
		return
	}

	cueCtx := cuecontext.New()
	val := cueCtx.CompileString(input)
	if err := val.Err(); err != nil {
		resp.Error = function.NewArgumentFuncError(0, "Invalid CUE: "+strings.TrimSpace(errors.Details(err, nil)))
		return
	}

	for i, s := range extra {
		other := cueCtx.CompileString(s)
		if err := other.Err(); err != nil {
			resp.Error = function.NewArgumentFuncError(int64(i+1), fmt.Sprintf("Invalid CUE in argument %d: %s", i+2, strings.TrimSpace(errors.Details(err, nil))))
			return
		}
		val = val.Unify(other)
	}

	if err := val.Err(); err != nil {
		resp.Error = function.NewFuncError("CUE unification error: " + strings.TrimSpace(errors.Details(err, nil)))
		return
	}

	if err := val.Validate(cue.Concrete(true)); err != nil {
		resp.Error = function.NewFuncError("CUE value is not concrete: " + strings.TrimSpace(errors.Details(err, nil)))
		return
	}

	tfVal, diags := cueValueToTerraform(val)
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		return
	}

	resp.Error = resp.Result.Set(ctx, types.DynamicValue(tfVal))
}
