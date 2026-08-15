package kwork

import (
	"testing"
)

func TestParseProject_Budget(t *testing.T) {
	testCases := []struct {
		name               string
		input              map[string]interface{}
		expectedBudgetFrom float64
		expectedBudgetTo   float64
		expectedValidFrom  bool
		expectedValidTo    bool
	}{
		{
			name: "Float numeric budget",
			input: map[string]interface{}{
				"id":                   float64(123),
				"price":                float64(2000),
				"possible_price_limit": float64(6000),
				"currency":             "RUB",
			},
			expectedBudgetFrom: 2000,
			expectedBudgetTo:   6000,
			expectedValidFrom:  true,
			expectedValidTo:    true,
		},
		{
			name: "Integer numeric budget",
			input: map[string]interface{}{
				"id":                   float64(124),
				"price":                2000,
				"possible_price_limit": 6000,
				"currency":             "RUB",
			},
			expectedBudgetFrom: 2000,
			expectedBudgetTo:   6000,
			expectedValidFrom:  true,
			expectedValidTo:    true,
		},
		{
			name: "String numeric budget",
			input: map[string]interface{}{
				"id":                   float64(125),
				"price":                "2000",
				"possible_price_limit": "6000",
			},
			expectedBudgetFrom: 2000,
			expectedBudgetTo:   6000,
			expectedValidFrom:  true,
			expectedValidTo:    true,
		},
		{
			name: "Missing possible_price_limit",
			input: map[string]interface{}{
				"id":    float64(126),
				"price": float64(2000),
			},
			expectedBudgetFrom: 2000,
			expectedBudgetTo:   0,
			expectedValidFrom:  true,
			expectedValidTo:    false,
		},
		{
			name: "Missing all budget fields",
			input: map[string]interface{}{
				"id": float64(127),
			},
			expectedBudgetFrom: 0,
			expectedBudgetTo:   0,
			expectedValidFrom:  false,
			expectedValidTo:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			proj := ParseProject(tc.input)
			
			if proj.BudgetFrom.Float64 != tc.expectedBudgetFrom {
				t.Errorf("Expected BudgetFrom %.2f, got %.2f", tc.expectedBudgetFrom, proj.BudgetFrom.Float64)
			}
			if proj.BudgetFrom.Valid != tc.expectedValidFrom {
				t.Errorf("Expected BudgetFrom.Valid %v, got %v", tc.expectedValidFrom, proj.BudgetFrom.Valid)
			}

			if proj.BudgetTo.Float64 != tc.expectedBudgetTo {
				t.Errorf("Expected BudgetTo %.2f, got %.2f", tc.expectedBudgetTo, proj.BudgetTo.Float64)
			}
			if proj.BudgetTo.Valid != tc.expectedValidTo {
				t.Errorf("Expected BudgetTo.Valid %v, got %v", tc.expectedValidTo, proj.BudgetTo.Valid)
			}
		})
	}
}

func TestParseProject_ExactBudget(t *testing.T) {
	raw := map[string]interface{}{
		"id": float64(123),
		"name": "Test",
		"price": float64(5000),
	}
	
	proj := ParseProject(raw)
	
	budget := proj.GetBudget()
	if budget.Type != "EXACT" {
		t.Errorf("expected EXACT budget, got %s", budget.Type)
	}
	if budget.Min != 5000 || budget.Max != 5000 {
		t.Errorf("expected 5000, got %f-%f", budget.Min, budget.Max)
	}
}
