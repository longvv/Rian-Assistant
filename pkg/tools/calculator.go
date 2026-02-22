package tools

// calculator.go - Safe math expression evaluator tool.
// Uses a pure-Go recursive descent parser — no external dependencies.
// Supports: +, -, *, /, %, parentheses, unary minus, and basic math functions.

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// CalculatorTool evaluates mathematical expressions safely without shell access.
type CalculatorTool struct{}

func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

func (t *CalculatorTool) Name() string { return "calculator" }
func (t *CalculatorTool) Description() string {
	return "Evaluate a mathematical expression and return the numeric result. Supports +, -, *, /, %, ^ (power), parentheses, and functions: sqrt, abs, floor, ceil, round, sin, cos, tan, log, log2, log10, exp, pi, e."
}

func (t *CalculatorTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "Mathematical expression to evaluate, e.g. '2 * (3 + 4) / 1.5' or 'sqrt(144) + pi'",
			},
		},
		"required": []string{"expression"},
	}
}

func (t *CalculatorTool) Execute(_ context.Context, args map[string]interface{}) *ToolResult {
	expr, ok := args["expression"].(string)
	if !ok || strings.TrimSpace(expr) == "" {
		return ErrorResult("expression is required")
	}

	result, err := evalExpr(strings.TrimSpace(expr))
	if err != nil {
		return ErrorResult(fmt.Sprintf("evaluation error: %v", err))
	}

	// Format nicely: integer if possible, otherwise float
	var formatted string
	if result == math.Trunc(result) && !math.IsInf(result, 0) {
		formatted = strconv.FormatInt(int64(result), 10)
	} else {
		formatted = strconv.FormatFloat(result, 'f', -1, 64)
	}

	answer := fmt.Sprintf("%s = %s", expr, formatted)
	return &ToolResult{
		ForLLM:  answer,
		ForUser: answer,
	}
}

// ─── Recursive descent parser ──────────────────────────────────────────────

type parser struct {
	input []rune
	pos   int
}

func evalExpr(s string) (float64, error) {
	p := &parser{input: []rune(s), pos: 0}
	result, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skipSpaces()
	if p.pos != len(p.input) {
		return 0, fmt.Errorf("unexpected character %q at position %d", string(p.input[p.pos:]), p.pos)
	}
	return result, nil
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.input) && unicode.IsSpace(p.input[p.pos]) {
		p.pos++
	}
}

func (p *parser) peek() (rune, bool) {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return 0, false
	}
	return p.input[p.pos], true
}

// parseExpr handles + and -
func (p *parser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		ch, ok := p.peek()
		if !ok || (ch != '+' && ch != '-') {
			break
		}
		p.pos++
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if ch == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

// parseTerm handles *, /, %
func (p *parser) parseTerm() (float64, error) {
	left, err := p.parsePower()
	if err != nil {
		return 0, err
	}
	for {
		ch, ok := p.peek()
		if !ok || (ch != '*' && ch != '/' && ch != '%') {
			break
		}
		p.pos++
		right, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		switch ch {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		case '%':
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			left = math.Mod(left, right)
		}
	}
	return left, nil
}

// parsePower handles ^ (right-associative)
func (p *parser) parsePower() (float64, error) {
	base, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	ch, ok := p.peek()
	if !ok || ch != '^' {
		return base, nil
	}
	p.pos++
	exp, err := p.parsePower() // right-associative
	if err != nil {
		return 0, err
	}
	return math.Pow(base, exp), nil
}

// parseUnary handles unary minus
func (p *parser) parseUnary() (float64, error) {
	ch, ok := p.peek()
	if ok && ch == '-' {
		p.pos++
		v, err := p.parseUnary()
		return -v, err
	}
	if ok && ch == '+' {
		p.pos++
		return p.parseUnary()
	}
	return p.parsePrimary()
}

// parsePrimary handles numbers, parentheses, and named functions/constants
func (p *parser) parsePrimary() (float64, error) {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	ch := p.input[p.pos]

	// Parenthesized sub-expression
	if ch == '(' {
		p.pos++
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipSpaces()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return 0, fmt.Errorf("expected closing ')'")
		}
		p.pos++
		return v, nil
	}

	// Named constant or function
	if unicode.IsLetter(ch) {
		return p.parseNamedToken()
	}

	// Number literal
	if unicode.IsDigit(ch) || ch == '.' {
		return p.parseNumber()
	}

	return 0, fmt.Errorf("unexpected character %q", string(ch))
}

func (p *parser) parseNumber() (float64, error) {
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsDigit(p.input[p.pos]) || p.input[p.pos] == '.' || p.input[p.pos] == 'e' || p.input[p.pos] == 'E') {
		// handle exponent sign: 1e-3, 2.5E+10
		if (p.input[p.pos] == 'e' || p.input[p.pos] == 'E') && p.pos+1 < len(p.input) &&
			(p.input[p.pos+1] == '+' || p.input[p.pos+1] == '-') {
			p.pos += 2
			continue
		}
		p.pos++
	}
	num := string(p.input[start:p.pos])
	v, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", num)
	}
	return v, nil
}

func (p *parser) parseNamedToken() (float64, error) {
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsLetter(p.input[p.pos]) || unicode.IsDigit(p.input[p.pos]) || p.input[p.pos] == '_') {
		p.pos++
	}
	name := strings.ToLower(string(p.input[start:p.pos]))

	// Constants
	switch name {
	case "pi":
		return math.Pi, nil
	case "e":
		return math.E, nil
	case "phi":
		return 1.6180339887498948482, nil
	}

	// Functions — expect '(' arg ')'
	p.skipSpaces()
	if p.pos >= len(p.input) || p.input[p.pos] != '(' {
		return 0, fmt.Errorf("unknown identifier %q (did you mean a function? add parentheses)", name)
	}
	p.pos++ // consume '('
	arg, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	p.skipSpaces()
	if p.pos >= len(p.input) || p.input[p.pos] != ')' {
		return 0, fmt.Errorf("expected ')' after function argument for %q", name)
	}
	p.pos++ // consume ')'

	switch name {
	case "sqrt":
		if arg < 0 {
			return 0, fmt.Errorf("sqrt of negative number")
		}
		return math.Sqrt(arg), nil
	case "abs":
		return math.Abs(arg), nil
	case "floor":
		return math.Floor(arg), nil
	case "ceil":
		return math.Ceil(arg), nil
	case "round":
		return math.Round(arg), nil
	case "sin":
		return math.Sin(arg), nil
	case "cos":
		return math.Cos(arg), nil
	case "tan":
		return math.Tan(arg), nil
	case "log", "ln":
		if arg <= 0 {
			return 0, fmt.Errorf("log of non-positive number")
		}
		return math.Log(arg), nil
	case "log2":
		if arg <= 0 {
			return 0, fmt.Errorf("log2 of non-positive number")
		}
		return math.Log2(arg), nil
	case "log10":
		if arg <= 0 {
			return 0, fmt.Errorf("log10 of non-positive number")
		}
		return math.Log10(arg), nil
	case "exp":
		return math.Exp(arg), nil
	}

	return 0, fmt.Errorf("unknown function %q", name)
}
