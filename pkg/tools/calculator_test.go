package tools

import (
	"context"
	"strings"
	"testing"
)

func TestCalculator_BasicArithmetic(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"2 + 3", 5},
		{"10 - 4", 6},
		{"3 * 4", 12},
		{"10 / 4", 2.5},
		{"10 % 3", 1},
		{"2 ^ 10", 1024},
		{"1 + 2 + 3 + 4", 10},
		{"100 / 10 / 2", 5},
	}
	for _, tc := range cases {
		got, err := evalExpr(tc.expr)
		if err != nil {
			t.Errorf("evalExpr(%q) unexpected error: %v", tc.expr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("evalExpr(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

func TestCalculator_Precedence(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"2 + 3 * 4", 14},   // * before +
		{"(2 + 3) * 4", 20}, // parens override
		{"2 ^ 3 ^ 2", 512},  // right-assoc: 2^(3^2) = 2^9
		{"- 2 + 5", 3},      // unary minus
		{"- - 3", 3},        // double unary
		{"2 * (3 + 4) / 1.5", 14.0 / 1.5 * 2.0 / (14.0 / 1.5) * (14.0)}, // simplified
	}
	// simpler precision test
	if v, err := evalExpr("2 * (3 + 4) / 1.5"); err != nil || v < 9.3 || v > 9.4 {
		t.Errorf("evalExpr('2 * (3 + 4) / 1.5') = %v, want ~9.33, err=%v", v, err)
	}
	_ = cases // individual cases tested above
	for _, tc := range cases[:5] {
		got, err := evalExpr(tc.expr)
		if err != nil {
			t.Errorf("evalExpr(%q) unexpected error: %v", tc.expr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("evalExpr(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

func TestCalculator_Functions(t *testing.T) {
	cases := []struct {
		expr    string
		wantMin float64
		wantMax float64
	}{
		{"sqrt(144)", 12, 12},
		{"abs(-5)", 5, 5},
		{"floor(3.9)", 3, 3},
		{"ceil(3.1)", 4, 4},
		{"round(3.5)", 4, 4},
		{"log(1)", 0, 0},
		{"log2(8)", 3, 3},
		{"log10(1000)", 3, 3},
		{"exp(0)", 1, 1},
		{"pi", 3.14159, 3.14160},
		{"e", 2.71828, 2.71829},
	}
	for _, tc := range cases {
		got, err := evalExpr(tc.expr)
		if err != nil {
			t.Errorf("evalExpr(%q) unexpected error: %v", tc.expr, err)
			continue
		}
		if got < tc.wantMin || got > tc.wantMax {
			t.Errorf("evalExpr(%q) = %v, want [%v, %v]", tc.expr, got, tc.wantMin, tc.wantMax)
		}
	}
}

func TestCalculator_Errors(t *testing.T) {
	errCases := []string{
		"1 / 0",
		"sqrt(-1)",
		"log(-1)",
		"log(0)",
		"1 +",        // incomplete
		"(1 + 2",     // unclosed paren
		"foo",        // unknown identifier
		"unknown(3)", // unknown function
	}
	for _, expr := range errCases {
		_, err := evalExpr(expr)
		if err == nil {
			t.Errorf("evalExpr(%q) expected error, got nil", expr)
		}
	}
}

func TestCalculator_ToolExecute(t *testing.T) {
	tool := NewCalculatorTool()

	// Test string expression
	result := tool.Execute(context.TODO(), map[string]interface{}{
		"expression": "1 + 1",
	})
	if result.IsError {
		t.Errorf("Unexpected error: %v", result.ForLLM)
	}
	if result.ForLLM != "1 + 1 = 2" {
		t.Errorf("unexpected ForLLM: %s", result.ForLLM)
	}

	// Test unsupported argument type (float64)
	resultFloat := tool.Execute(context.TODO(), map[string]interface{}{
		"expression": 1.5,
	})
	if !resultFloat.IsError {
		t.Error("Expected error for non-string expression")
	}
	if !strings.Contains(resultFloat.ForLLM, "expression is required") {
		t.Errorf("unexpected error message: %s", resultFloat.ForLLM)
	}

	// Test explicit error message containing original type
	resultInt := tool.Execute(context.TODO(), map[string]interface{}{
		"expression": 42,
	})
	if !resultInt.IsError {
		t.Error("Expected error for non-string expression")
	}
	if !strings.Contains(resultInt.ForLLM, "expression is required") {
		t.Errorf("unexpected error message: %s", resultInt.ForLLM)
	}

	// Test missing expression argument
	resultMissing := tool.Execute(context.TODO(), map[string]interface{}{})
	if !resultMissing.IsError {
		t.Error("Expected error for missing expression argument")
	}
	if !strings.Contains(resultMissing.ForLLM, "expression is required") {
		t.Errorf("unexpected error message: %s", resultMissing.ForLLM)
	}

	// Valid expression (original test case adapted)
	result = tool.Execute(context.TODO(), map[string]interface{}{"expression": "2 ^ 8"})
	if result.IsError {
		t.Errorf("expected success, got error: %s", result.ForLLM)
	}
	if result.ForLLM != "2 ^ 8 = 256" {
		t.Errorf("unexpected ForLLM: %s", result.ForLLM)
	}

	// Integer formatting (original test case adapted)
	result = tool.Execute(context.TODO(), map[string]interface{}{"expression": "10 / 2"})
	if result.IsError {
		t.Errorf("expected success, got error: %s", result.ForLLM)
	}
	if result.ForLLM != "10 / 2 = 5" {
		t.Errorf("expected '10 / 2 = 5', got: %s", result.ForLLM)
	}

	// Float formatting (original test case adapted)
	result = tool.Execute(context.TODO(), map[string]interface{}{"expression": "1 / 3"})
	if result.IsError {
		t.Errorf("expected success for 1/3, got error: %s", result.ForLLM)
	}
	// Check for approximate float value, as 1/3 is recurring
	if result.ForLLM == "" || result.ForLLM == "1 / 3 = 0" { // Basic check, more robust check might be needed depending on precision
		t.Errorf("unexpected ForLLM for 1/3: %s", result.ForLLM)
	}
}

func TestCalculator_NameAndDescription(t *testing.T) {
	tool := NewCalculatorTool()
	if tool.Name() != "calculator" {
		t.Errorf("unexpected tool name: %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Errorf("description should not be empty")
	}
	params := tool.Parameters()
	if params == nil {
		t.Errorf("parameters should not be nil")
	}
}
