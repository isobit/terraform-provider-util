package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestCueDecodeFunction_Metadata(t *testing.T) {
	f := &CueDecodeFunction{}
	resp := &function.MetadataResponse{}
	f.Metadata(context.Background(), function.MetadataRequest{}, resp)
	if resp.Name != "cuedecode" {
		t.Fatalf("expected name 'cuedecode', got %q", resp.Name)
	}
}

func TestCueDecodeFunction_Definition(t *testing.T) {
	f := &CueDecodeFunction{}
	resp := &function.DefinitionResponse{}
	f.Definition(context.Background(), function.DefinitionRequest{}, resp)
	if len(resp.Definition.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(resp.Definition.Parameters))
	}
	if resp.Definition.VariadicParameter == nil {
		t.Fatal("expected variadic parameter to be set")
	}
	if resp.Definition.Return == nil {
		t.Fatal("expected return type to be set")
	}
}

// buildCueDecodeArgs constructs arguments data for cuedecode tests.
// The first arg is the primary CUE string param, the rest are variadic.
func buildCueDecodeArgs(args ...string) function.ArgumentsData {
	variadicTypes := make([]attr.Type, len(args)-1)
	variadicValues := make([]attr.Value, len(args)-1)
	for i := 1; i < len(args); i++ {
		variadicTypes[i-1] = basetypes.StringType{}
		variadicValues[i-1] = types.StringValue(args[i])
	}
	return function.NewArgumentsData([]attr.Value{
		types.StringValue(args[0]),
		basetypes.NewTupleValueMust(variadicTypes, variadicValues),
	})
}

func runCueDecode(t *testing.T, args ...string) (*function.RunResponse, *function.FuncError) {
	t.Helper()
	f := &CueDecodeFunction{}
	ctx := context.Background()

	defResp := &function.DefinitionResponse{}
	f.Definition(ctx, function.DefinitionRequest{}, defResp)

	result, err := defResp.Definition.Return.NewResultData(ctx)
	if err != nil {
		t.Fatal(err)
	}

	resp := &function.RunResponse{Result: result}
	req := function.RunRequest{
		Arguments: buildCueDecodeArgs(args...),
	}
	f.Run(ctx, req, resp)
	return resp, resp.Error
}

func TestCueDecodeFunction_Run_String(t *testing.T) {
	_, err := runCueDecode(t, `"hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Text)
	}
}

func TestCueDecodeFunction_Run_Struct(t *testing.T) {
	_, err := runCueDecode(t, `{name: "test", port: 8080}`)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Text)
	}
}

func TestCueDecodeFunction_Run_Invalid(t *testing.T) {
	_, err := runCueDecode(t, `{invalid:`)
	if err == nil {
		t.Fatal("expected error for invalid CUE")
	}
}

func TestCueDecodeFunction_Run_NonConcrete(t *testing.T) {
	_, err := runCueDecode(t, `{name: string}`)
	if err == nil {
		t.Fatal("expected error for non-concrete value")
	}
}

func TestCueDecodeFunction_Run_Unify(t *testing.T) {
	_, err := runCueDecode(t,
		`{name: "hello", port: 8080}`,
		`{name: =~"^[a-z]+$", port: >1024}`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Text)
	}
}

func TestCueDecodeFunction_Run_UnifyFailure(t *testing.T) {
	_, err := runCueDecode(t,
		`{name: "Hello", port: 80}`,
		`{name: =~"^[a-z]+$", port: >1024}`,
	)
	if err == nil {
		t.Fatal("expected error for constraint violation")
	}
	if !strings.Contains(err.Text, "invalid value") {
		t.Fatalf("expected constraint error details, got: %s", err.Text)
	}
}

func TestCueDecodeFunction_Run_UnifyMultiple(t *testing.T) {
	_, err := runCueDecode(t,
		`{name: "app", port: 9090, debug: true}`,
		`{port: >1024}`,
		`{debug: bool}`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Text)
	}
}

func TestCueDecodeFunction_Run_UnifyInvalidConstraint(t *testing.T) {
	_, err := runCueDecode(t,
		`{name: "hello"}`,
		`{bad syntax`,
	)
	if err == nil {
		t.Fatal("expected error for invalid constraint CUE")
	}
	if !strings.Contains(err.Text, "Invalid CUE in argument 2") {
		t.Fatalf("expected error to reference argument 2, got: %s", err.Text)
	}
}

func TestCueDecodeFunction_Run_UnifyMergeValues(t *testing.T) {
	_, err := runCueDecode(t,
		`{a: 1}`,
		`{b: 2}`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Text)
	}
}

func TestCueDecodeFunction_Run_UnifyConflict(t *testing.T) {
	_, err := runCueDecode(t,
		`{x: 1}`,
		`{x: 2}`,
	)
	if err == nil {
		t.Fatal("expected error for conflicting values")
	}
	if !strings.Contains(err.Text, "conflict") {
		t.Fatalf("expected conflict error, got: %s", err.Text)
	}
}
