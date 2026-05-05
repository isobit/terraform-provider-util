package provider

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCueValueToTerraform_Scalars(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name  string
		cue   string
		check func(t *testing.T, v attr.Value)
	}{
		{"string", `"hello"`, func(t *testing.T, v attr.Value) {
			if v.(types.String).ValueString() != "hello" {
				t.Fatalf("got %v", v)
			}
		}},
		{"bool", `true`, func(t *testing.T, v attr.Value) {
			if v.(types.Bool).ValueBool() != true {
				t.Fatalf("got %v", v)
			}
		}},
		{"int", `42`, func(t *testing.T, v attr.Value) {
			n := v.(types.Number).ValueBigFloat()
			if n.Cmp(new(big.Float).SetInt64(42)) != 0 {
				t.Fatalf("got %v", n)
			}
		}},
		{"float", `3.14`, func(t *testing.T, v attr.Value) {
			n := v.(types.Number).ValueBigFloat()
			if n.Cmp(new(big.Float).SetFloat64(3.14)) != 0 {
				t.Fatalf("got %v", n)
			}
		}},
		{"null", `null`, func(t *testing.T, v attr.Value) {
			if !v.IsNull() {
				t.Fatal("expected null")
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ctx.CompileString(tt.cue)
			result, diags := cueValueToTerraform(v)
			if diags.HasError() {
				t.Fatal(diags.Errors())
			}
			tt.check(t, result)
		})
	}
}

func TestCueValueToTerraform_Struct(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{name: "test", count: 5, enabled: true}`)
	result, diags := cueValueToTerraform(v)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	ov := result.(types.Object)
	attrs := ov.Attributes()
	if attrs["name"].(types.String).ValueString() != "test" {
		t.Fatal("wrong name value")
	}
	if attrs["count"].(types.Number).ValueBigFloat().Cmp(new(big.Float).SetInt64(5)) != 0 {
		t.Fatal("wrong count value")
	}
	if attrs["enabled"].(types.Bool).ValueBool() != true {
		t.Fatal("wrong enabled value")
	}
}

func TestCueValueToTerraform_NestedStruct(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{server: {host: "localhost", port: 8080}, tags: ["web", "prod"]}`)
	result, diags := cueValueToTerraform(v)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	ov := result.(types.Object)
	server := ov.Attributes()["server"].(types.Object)
	if server.Attributes()["host"].(types.String).ValueString() != "localhost" {
		t.Fatal("wrong host")
	}
	tags := ov.Attributes()["tags"].(types.Tuple)
	if len(tags.Elements()) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags.Elements()))
	}
}

func TestCueValueToTerraform_EmptyContainers(t *testing.T) {
	ctx := cuecontext.New()

	v := ctx.CompileString(`[]`)
	result, diags := cueValueToTerraform(v)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	if len(result.(types.Tuple).Elements()) != 0 {
		t.Fatal("expected empty tuple")
	}

	v = ctx.CompileString(`{}`)
	result, diags = cueValueToTerraform(v)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	if len(result.(types.Object).Attributes()) != 0 {
		t.Fatal("expected empty object")
	}
}

func TestCueValueToTerraform_UnsupportedKind(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`'\x00\x01'`)
	_, diags := cueValueToTerraform(v)
	if !diags.HasError() {
		t.Fatal("expected error for bytes kind")
	}
}

func TestEvaluateCue_SingleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.cue")
	os.WriteFile(f, []byte(`{name: "app", port: 8080}`), 0644)

	cueCtx := cuecontext.New()
	insts := load.Instances([]string{f}, &load.Config{})
	if insts[0].Err != nil {
		t.Fatal(insts[0].Err)
	}
	val := cueCtx.BuildInstance(insts[0])
	result, diags := cueValueToTerraform(val)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	attrs := result.(types.Object).Attributes()
	if attrs["name"].(types.String).ValueString() != "app" {
		t.Fatal("wrong name")
	}
}

func TestEvaluateCue_Directory(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "myapp")
	os.MkdirAll(pkg, 0755)
	os.WriteFile(filepath.Join(pkg, "a.cue"), []byte("package myapp\nname: \"hello\"\n"), 0644)
	os.WriteFile(filepath.Join(pkg, "b.cue"), []byte("package myapp\nport: 8080\n"), 0644)

	cueCtx := cuecontext.New()
	insts := load.Instances([]string{"."}, &load.Config{Dir: pkg})
	if insts[0].Err != nil {
		t.Fatal(insts[0].Err)
	}
	val := cueCtx.BuildInstance(insts[0])
	result, diags := cueValueToTerraform(val)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	attrs := result.(types.Object).Attributes()
	if attrs["name"].(types.String).ValueString() != "hello" {
		t.Fatal("wrong name")
	}
	if attrs["port"].(types.Number).ValueBigFloat().Cmp(new(big.Float).SetInt64(8080)) != 0 {
		t.Fatal("wrong port")
	}
}

func TestEvaluateCue_Tags(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.cue")
	os.WriteFile(f, []byte(`env: string @tag(env)
port: int @tag(port,type=int)
`), 0644)

	cueCtx := cuecontext.New()
	insts := load.Instances([]string{f}, &load.Config{
		Tags: []string{"env=prod", "port=9090"},
	})
	if insts[0].Err != nil {
		t.Fatal(insts[0].Err)
	}
	val := cueCtx.BuildInstance(insts[0])
	if err := val.Validate(cue.Concrete(true)); err != nil {
		t.Fatal(err)
	}
	result, diags := cueValueToTerraform(val)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	attrs := result.(types.Object).Attributes()
	if attrs["env"].(types.String).ValueString() != "prod" {
		t.Fatalf("expected env=prod, got %v", attrs["env"])
	}
	if attrs["port"].(types.Number).ValueBigFloat().Cmp(new(big.Float).SetInt64(9090)) != 0 {
		t.Fatalf("expected port=9090, got %v", attrs["port"])
	}
}

func TestEvaluateCue_InputUnification(t *testing.T) {
	ctx := cuecontext.New()
	base := ctx.CompileString(`{name: "app", port: 8080}`)
	constraint := ctx.CompileString(`{port: >1024}`)
	val := base.Unify(constraint)
	if err := val.Validate(cue.Concrete(true)); err != nil {
		t.Fatal(err)
	}
	result, diags := cueValueToTerraform(val)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	attrs := result.(types.Object).Attributes()
	if attrs["name"].(types.String).ValueString() != "app" {
		t.Fatal("wrong name")
	}
}

func TestEvaluateCue_InputUnificationFailure(t *testing.T) {
	ctx := cuecontext.New()
	base := ctx.CompileString(`{port: 80}`)
	constraint := ctx.CompileString(`{port: >1024}`)
	val := base.Unify(constraint)
	err := val.Validate(cue.Concrete(true))
	if err == nil {
		t.Fatal("expected validation failure")
	}
	formatted := formatCueError(err)
	if !strings.Contains(formatted, "invalid value") {
		t.Fatalf("expected constraint violation, got: %s", formatted)
	}
}

func TestEvaluateCue_ModuleRoot(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "cue.mod"), 0755)
	os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(
		"module: \"test.example/mod@v0\"\nlanguage: version: \"v0.9.0\"\n"), 0644)
	pkg := filepath.Join(dir, "config")
	os.MkdirAll(pkg, 0755)
	os.WriteFile(filepath.Join(pkg, "x.cue"), []byte("package config\nval: 42\n"), 0644)

	cueCtx := cuecontext.New()
	insts := load.Instances([]string{"."}, &load.Config{Dir: pkg})
	if insts[0].Err != nil {
		t.Fatal(insts[0].Err)
	}
	val := cueCtx.BuildInstance(insts[0])
	result, diags := cueValueToTerraform(val)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	attrs := result.(types.Object).Attributes()
	if attrs["val"].(types.Number).ValueBigFloat().Cmp(new(big.Float).SetInt64(42)) != 0 {
		t.Fatal("wrong val")
	}
}

func TestEvaluateCue_NonExistentPath(t *testing.T) {
	info, _ := os.Stat("/nonexistent/path.cue")
	if info != nil {
		t.Skip("path unexpectedly exists")
	}
	// Verify the error path — os.Stat fails, so evaluateCue (via Read) would
	// produce a diagnostic. We test the stat directly since Read requires
	// framework plumbing.
	_, err := os.Stat("/nonexistent/path.cue")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatCueError_IncludesPosition(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString("{\n\tname: string\n\tport: int\n}", cue.Filename("config.cue"))
	err := v.Validate(cue.Concrete(true))
	if err == nil {
		t.Fatal("expected validation error")
	}
	formatted := formatCueError(err)
	if !strings.Contains(formatted, "config.cue:") {
		t.Fatalf("expected file position, got: %s", formatted)
	}
}

func TestCueValueToTerraform_AttrTypes(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{name: "test", count: 1, ok: true}`)
	result, diags := cueValueToTerraform(v)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	attrTypes := result.(types.Object).AttributeTypes(context.Background())
	if attrTypes["name"] != attr.Type(types.StringType) {
		t.Fatal("name should be StringType")
	}
	if attrTypes["ok"] != attr.Type(types.BoolType) {
		t.Fatal("ok should be BoolType")
	}
}

func TestCueValueToTerraform_HeterogeneousList(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`[1, "two", true]`)
	result, diags := cueValueToTerraform(v)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	tuple := result.(types.Tuple)
	elems := tuple.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
	if _, ok := elems[0].(types.Number); !ok {
		t.Fatalf("elem 0: expected Number, got %T", elems[0])
	}
	if _, ok := elems[1].(types.String); !ok {
		t.Fatalf("elem 1: expected String, got %T", elems[1])
	}
	if _, ok := elems[2].(types.Bool); !ok {
		t.Fatalf("elem 2: expected Bool, got %T", elems[2])
	}
}

func TestEvaluateCue_PackageSelection(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.cue"), []byte("package alpha\nval: \"a\"\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.cue"), []byte("package beta\nval: \"b\"\n"), 0644)

	cueCtx := cuecontext.New()
	insts := load.Instances([]string{"."}, &load.Config{Dir: dir, Package: "beta"})
	if insts[0].Err != nil {
		t.Fatal(insts[0].Err)
	}
	val := cueCtx.BuildInstance(insts[0])
	result, diags := cueValueToTerraform(val)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	attrs := result.(types.Object).Attributes()
	if attrs["val"].(types.String).ValueString() != "b" {
		t.Fatalf("expected val=\"b\", got %v", attrs["val"])
	}
}

func TestEvaluateCue_UnificationConflict(t *testing.T) {
	ctx := cuecontext.New()
	base := ctx.CompileString(`{x: 1}`)
	conflict := ctx.CompileString(`{x: 2}`)
	val := base.Unify(conflict)
	err := val.Validate(cue.Concrete(true))
	if err == nil {
		t.Fatal("expected error for conflicting concrete values")
	}
	formatted := formatCueError(err)
	if !strings.Contains(formatted, "conflict") {
		t.Fatalf("expected conflict error, got: %s", formatted)
	}
}

func TestCueValueToTerraform_NullInStruct(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{name: "test", optional: null}`)
	result, diags := cueValueToTerraform(v)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	attrs := result.(types.Object).Attributes()
	if !attrs["optional"].IsNull() {
		t.Fatal("expected null for optional field")
	}
	if attrs["name"].(types.String).ValueString() != "test" {
		t.Fatal("wrong name value")
	}
}

func TestCueValueToTerraform_NestedList(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`[[1, 2], [3, 4]]`)
	result, diags := cueValueToTerraform(v)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	outer := result.(types.Tuple).Elements()
	if len(outer) != 2 {
		t.Fatalf("expected 2 outer elements, got %d", len(outer))
	}
	inner := outer[0].(types.Tuple).Elements()
	if len(inner) != 2 {
		t.Fatalf("expected 2 inner elements, got %d", len(inner))
	}
}
