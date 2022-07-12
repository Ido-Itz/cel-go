// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repl

import (
	"testing"

	"github.com/google/cel-go/cel"

	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

var testTextDescriptorFile string = "testdata/attribute_context_fds.textproto"

func mustParseType(t testing.TB, name string) *exprpb.Type {
	t.Helper()
	ty, err := ParseType(name)
	if err != nil {
		t.Fatalf("ParseType(%s) failed", name)
	}
	return ty
}

func TestEvalSimple(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	_, _, err = eval.Evaluate("[1, 2, 3]")

	if err != nil {
		t.Errorf("eval.Evaluate('[1, 2, 3]') got %v, wanted non-error", err)
	}
}

func TestEvalSingleLetVar(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetVar("x", "2 + 2", nil)
	if err != nil {
		t.Errorf("eval.AddLetVar('x', '2 + 2') got %v, wanted non-error", err)
	}

	_, _, err = eval.Evaluate("[1, 2, 3, x]")

	if err != nil {
		t.Errorf("eval.Evaluate('[1, 2, 3, x]') got %v, wanted non-error", err)
	}
}

func TestEvalMultiLet(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	eval.AddLetVar("x", "20/5", nil)
	eval.AddLetVar("y", "x * 3", nil)
	eval.AddLetVar("x", "20", nil)

	_, _, err = eval.Evaluate("[1, 2, 3, x, y]")
	if err != nil {
		t.Errorf("eval.Evaluate('[1, 2, 3, x, y]') got %v, wanted non-error", err)
	}
}

func TestEvalError(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	eval.AddLetVar("x", "1", nil)
	eval.AddLetVar("y", "0", nil)

	_, _, err = eval.Evaluate("x / y")
	if err == nil {
		t.Errorf("eval.Evaluate('x / y') got non-error, wanted division by zero")
	}
}

func TestLetError(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetVar("y", "z + 1", nil)
	if err == nil {
		t.Errorf("eval.AddLetVar('y', 'z + 1') got %v, wanted error", err)
	}

	result, _, err := eval.Evaluate("y")
	if err == nil {
		t.Errorf("eval.Evaluate('y') got result %v, wanted error", result.Value())
	}
}

func TestLetTypeHintError(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetVar("y", "10u", mustParseType(t, "int"))
	if err == nil {
		t.Errorf("eval.AddLetVar('y', '10u') got %v, wanted error", err)
	}

	result, _, err := eval.Evaluate("y")
	if err == nil {
		t.Errorf("eval.Evaluate('y') got result %v, wanted error", result)
	}
}

func TestDeclareError(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	eval.AddDeclVar("z", mustParseType(t, "double"))
	eval.AddLetVar("y", "z + 10.0", nil)
	err = eval.AddLetVar("z", "'2.0'", nil)
	if err == nil {
		t.Errorf("eval.AddLetVar('z', '\"2.0\"') got %v, wanted error", err)
	}

	err = eval.AddDeclVar("z", mustParseType(t, "string"))
	if err == nil {
		t.Errorf("eval.AddDeclVar('z', string) got %v, wanted error", err)
	}
}

func TestDelError(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	eval.AddLetVar("z", "41", nil)
	eval.AddLetVar("y", "z + 1", nil)

	err = eval.DelLetVar("z")
	if err == nil {
		t.Errorf("eval.DelLetVar('z') got %v, wanted error", err)
	}

	val, _, err := eval.Evaluate("y")
	if err != nil {
		t.Errorf("eval.Evaluate('y') failed %v, wanted non-error", err)
	} else if val.Value().(int64) != 42 {
		t.Errorf("eval.Evaluate('y') got %v, wanted 42", val.Value())
	}

}

func TestAddLetFn(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	eval.AddLetFn("fn", []letFunctionParam{
		{identifier: "x", typeHint: mustParseType(t, "int")},
		{identifier: "y", typeHint: mustParseType(t, "int")}},
		mustParseType(t, "int"),
		"x * x - y * y")

	eval.AddLetVar("testcases", "[[1, 2], [2, 3], [3, 4], [10, 20]]", mustParseType(t, "list(list(int))"))

	result, _, err := eval.Evaluate("testcases.all(e, fn(e[0], e[1]) == (e[0] - e[1]) * (e[0] + e[1]))")

	if err != nil {
		t.Errorf("eval.Evaluate() got error %v wanted nil", err)
	} else if !result.Value().(bool) {
		t.Errorf("eval.Evaluate() got %v wanted true", result.Value())
	}
}

func TestAddLetFnComposed(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetFn("square", []letFunctionParam{{
		identifier: "x",
		typeHint:   mustParseType(t, "int"),
	}}, mustParseType(t, "int"), "x * x")

	if err != nil {
		t.Errorf("eval.AddLetFn(square x -> int) got error %v expected nil", err)
	}

	err = eval.AddLetFn("squareDiff", []letFunctionParam{
		{
			identifier: "x",
			typeHint:   mustParseType(t, "int"),
		},
		{
			identifier: "y",
			typeHint:   mustParseType(t, "int"),
		}}, mustParseType(t, "int"), "square(x) - square(y)")

	if err != nil {
		t.Errorf("eval.AddLetFn(squareDiff x, y -> int) got error %v expected nil", err)
	}

	result, _, err := eval.Evaluate("squareDiff(4, 3)")

	if err != nil {
		t.Errorf("eval.Evaluate() got error %v wanted nil", err)
	} else if result.Value().(int64) != 7 {
		t.Errorf("eval.Evaluate() got %v wanted true", result.Value())
	}
}

func TestAddLetFnErrorOnTypeChange(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetFn("square", []letFunctionParam{
		{identifier: "x", typeHint: mustParseType(t, "int")}},
		mustParseType(t, "int"),
		"x * x")

	if err != nil {
		t.Errorf("eval.AddLetFn('square x -> int') got error %v expected nil", err)
	}

	eval.AddLetVar("y", "square(1) + 1", nil)

	// Overloads not yet supported
	err = eval.AddLetFn("square", []letFunctionParam{
		{identifier: "x", typeHint: mustParseType(t, "double")}},
		mustParseType(t, "double"),
		"x * x")

	if err == nil {
		t.Error("eval.AddLetFn('square x -> double') got nil, expected error")
	}

	result, _, err := eval.Evaluate("y")

	if err != nil {
		t.Errorf("eval.Evaluate() got error %v wanted nil", err)
	} else if result.Value().(int64) != 2 {
		t.Errorf("eval.Evaluate() got %v wanted true", result.Value())
	}
}

func TestAddLetFnErrorTypeMismatch(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetFn("fn", []letFunctionParam{
		{identifier: "x", typeHint: mustParseType(t, "int")},
		{identifier: "y", typeHint: mustParseType(t, "int")}},
		mustParseType(t, "double"),
		"x * x - y * y")

	if err == nil {
		t.Error("eval.AddLetFn('fn x : int, y : int -> double') got nil expected error")
	}
}

func TestAddLetFnErrorBadExpr(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetFn("fn", []letFunctionParam{
		{identifier: "y", typeHint: mustParseType(t, "string")}},
		mustParseType(t, "int"),
		"2 - y")

	if err == nil {
		t.Error("eval.AddLetFn('fn y : string -> int = 2 - y') got nil wanted error")
	}
}

func TestAddDeclFn(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddDeclFn("fn", []letFunctionParam{
		{identifier: "x", typeHint: mustParseType(t, "int")},
		{identifier: "y", typeHint: mustParseType(t, "int")}},
		mustParseType(t, "int"))

	if err != nil {
		t.Errorf("eval.AddDeclFn('fn(x:int, y:int): int') got error %v wanted nil", err)
	}

	err = eval.AddLetVar("z", "fn(1, 2)", nil)

	if err != nil {
		t.Errorf("eval.AddLetVar('z = fn(1, 2)') got error %v wanted nil", err)
	}
}

func TestAddDeclFnError(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddDeclFn("fn", []letFunctionParam{
		{identifier: "x", typeHint: mustParseType(t, "int")},
		{identifier: "y", typeHint: mustParseType(t, "int")}},
		mustParseType(t, "int"))

	if err != nil {
		t.Errorf("eval.AddDeclFn('fn(x:int, y:int): int') got error %v wanted nil", err)
	}

	err = eval.AddLetVar("z", "fn(1, 2)", nil)

	if err != nil {
		t.Errorf("eval.AddLetVar('z = fn(1, 2)') got error %v wanted nil", err)
	}

	err = eval.AddDeclFn("fn", []letFunctionParam{
		{identifier: "x", typeHint: mustParseType(t, "string")},
		{identifier: "y", typeHint: mustParseType(t, "string")}},
		mustParseType(t, "int"))

	if err == nil {
		t.Errorf("eval.AddDeclFn('fn(x:string, y:string): int') got nil wanted error")
	}

}

func TestDelLetFn(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetFn("fn", []letFunctionParam{
		{identifier: "y", typeHint: mustParseType(t, "int")}},
		mustParseType(t, "int"),
		"2 - y")

	if err != nil {
		t.Fatalf("eval.AddLetFn('fn (y : string): int -> 2 - y') failed: %s", err)
	}

	err = eval.DelLetFn("fn")
	if err != nil {
		t.Errorf("eval.DelLetFn('fn') got error %s, wanted nil", err)
	}

	err = eval.AddLetVar("x", "fn(2)", nil)
	if err == nil {
		t.Error("eval.AddLetVar('fn(2)') got nil, wanted error")
	}
}

func TestDelLetFnError(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetFn("fn", []letFunctionParam{
		{identifier: "y", typeHint: mustParseType(t, "int")}},
		mustParseType(t, "int"),
		"2 - y")

	if err != nil {
		t.Fatalf("eval.AddLetFn('fn (y : string): int -> 2 - y') failed: %s", err)
	}

	err = eval.AddLetVar("x", "fn(2)", nil)
	if err != nil {
		t.Errorf("eval.AddLetVar('fn(2)') got error %s, wanted nil", err)
	}

	err = eval.DelLetFn("fn")
	if err == nil {
		t.Error("eval.DelLetFn('fn') got nil, wanted error")
	}
}

func TestInstanceFunction(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddLetFn("int.plus", []letFunctionParam{
		{identifier: "y", typeHint: mustParseType(t, "int")}},
		mustParseType(t, "int"),
		"this + y")

	if err != nil {
		t.Fatalf("eval.AddLetFn('int.plus(y : int): int -> this + y') failed: %s", err)
	}

	val, _, err := eval.Evaluate("40.plus(2)")
	if err != nil {
		t.Errorf("eval.Eval('40.plus(2)') got error %v, wanted nil", err)
	}
	if val.Value().(int64) != 42 {
		t.Errorf("eval.Eval('40.plus(2)') got %s, wanted 42", val)
	}
}

type testContainerOption struct {
	container string
}

func (o *testContainerOption) String() string {
	return o.container
}

func (o *testContainerOption) Option() cel.EnvOption {
	return cel.Container(o.container)
}

func TestSetOption(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddOption(&testContainerOption{container: "google.protobuf"})
	if err != nil {
		t.Errorf("eval.AddOption() got error: %v, wanted nil", err)
	}

	val, _, err := eval.Evaluate("Int64Value{value: 42} == 42")
	if err != nil {
		t.Errorf("eval.Evaluate() got error: %v, expected nil", err)
	}

	if !val.Value().(bool) {
		t.Error("eval.Evaluate() got false expected true")
	}
}

func TestSetOptionError(t *testing.T) {
	eval, err := NewEvaluator()
	if err != nil {
		t.Fatalf("NewEvaluator() failed with: %v", err)
	}

	err = eval.AddOption(&testContainerOption{container: "google.protobuf"})
	if err != nil {
		t.Errorf("eval.AddOption() got error: %v, wanted nil", err)
	}

	err = eval.AddLetVar("x", "Int64Value{value: 42} == 42", nil)
	if err != nil {
		t.Errorf("eval.Evaluate() got error: %v, expected nil", err)
	}

	err = eval.AddOption(&testContainerOption{container: ""})
	if err == nil {
		t.Error("eval.AddOption() got nil expected error")
	}
}

func TestProcess(t *testing.T) {
	var testCases = []struct {
		name      string
		commands  []Cmder
		wantText  string
		wantExit  bool
		wantError bool
	}{
		{
			name: "OptionBasic",
			commands: []Cmder{
				&simpleCmd{
					cmd: "option",
					args: []string{
						"--container",
						"google.protobuf",
					},
				},
			},
			wantText:  "",
			wantExit:  false,
			wantError: false,
		},
		{
			name: "OptionContainer",
			commands: []Cmder{
				&simpleCmd{
					cmd: "option",
					args: []string{
						"--container",
						"google.protobuf",
					},
				},
				&evalCmd{
					expr: "Int64Value{value: 20}",
				},
			},
			wantText:  "20 : wrapper(int)",
			wantExit:  false,
			wantError: false,
		},
		{
			name: "LoadDescriptorsError",
			commands: []Cmder{
				&simpleCmd{
					cmd: "load_descriptors",
					args: []string{
						"<not a file>",
					},
				},
			},
			wantText:  "",
			wantExit:  false,
			wantError: true,
		},
		{
			name: "LoadDescriptorsAddsTypes",
			commands: []Cmder{
				&simpleCmd{
					cmd: "load_descriptors",
					args: []string{
						testTextDescriptorFile,
					},
				},
				&simpleCmd{
					cmd: "option",
					args: []string{
						"--container",
						"google.rpc.context",
					},
				},
				&evalCmd{
					expr: "AttributeContext.Request{host: 'www.example.com'}",
				},
			},
			wantText:  `host:"www.example.com" : google.rpc.context.AttributeContext.Request`,
			wantExit:  false,
			wantError: false,
		},
		{
			name: "Status",
			commands: []Cmder{
				&simpleCmd{
					cmd: "option",
					args: []string{
						"--container",
						"google",
					},
				},
				&letVarCmd{
					identifier: "x",
					typeHint:   mustParseType(t, "int"),
					src:        "1",
				},
				&letFnCmd{
					identifier: "fn",
					src:        "2",
					resultType: mustParseType(t, "int"),
					params:     nil,
				},
				&simpleCmd{
					cmd:  "status",
					args: []string{},
				},
			},
			wantText: `// Options
%option --container 'google'

// Functions
%let fn() : int -> 2

// Variables
%let x = 1
`,
			wantExit:  false,
			wantError: false,
		},
		{
			name: "Reset",
			commands: []Cmder{
				&simpleCmd{
					cmd: "option",
					args: []string{
						"--container",
						"google",
					},
				},
				&letVarCmd{
					identifier: "x",
					typeHint:   mustParseType(t, "int"),
					src:        "1",
				},
				&letFnCmd{
					identifier: "fn",
					src:        "2",
					resultType: mustParseType(t, "int"),
					params:     nil,
				},
				&simpleCmd{
					cmd:  "reset",
					args: nil,
				},
				&simpleCmd{
					cmd:  "status",
					args: []string{},
				},
			},
			wantText: `// Options

// Functions

// Variables
`,
			wantExit:  false,
			wantError: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			eval, err := NewEvaluator()
			if err != nil {
				t.Fatalf("NewEvaluator returned error: %v, wanted nil", err)
			}
			n := len(tc.commands)
			for _, cmd := range tc.commands[:n-1] {
				// only need output of last command
				eval.Process(cmd)
			}
			text, exit, err := eval.Process(tc.commands[n-1])

			gotErr := false
			if err != nil {
				gotErr = true
			}

			if text != tc.wantText || exit != tc.wantExit || (gotErr != tc.wantError) {
				t.Errorf("For command %s got (output: '%s' exit: %v err: %v (%v)) wanted (output: '%s' exit: %v err: %v)",
					tc.commands[n-1], text, exit, gotErr, err, tc.wantText, tc.wantExit, tc.wantError)
			}
		})
	}
}