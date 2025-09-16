package recommendations

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	assert.NotNil(t, client)
	assert.NotNil(t, client.costExplorerClient)
	assert.Equal(t, "us-east-1", client.region)
}

func TestConvertPaymentOption(t *testing.T) {
	tests := []struct {
		name     string
		option   string
		expected types.PaymentOption
	}{
		{
			name:     "all upfront",
			option:   "all-upfront",
			expected: types.PaymentOptionAllUpfront,
		},
		{
			name:     "partial upfront",
			option:   "partial-upfront",
			expected: types.PaymentOptionPartialUpfront,
		},
		{
			name:     "no upfront",
			option:   "no-upfront",
			expected: types.PaymentOptionNoUpfront,
		},
		{
			name:     "invalid option defaults to partial upfront",
			option:   "invalid",
			expected: types.PaymentOptionPartialUpfront,
		},
		{
			name:     "empty option defaults to partial upfront",
			option:   "",
			expected: types.PaymentOptionPartialUpfront,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertPaymentOption(tt.option)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertTermInYears(t *testing.T) {
	tests := []struct {
		name     string
		years    int
		expected types.TermInYears
	}{
		{
			name:     "1 year",
			years:    1,
			expected: types.TermInYearsOneYear,
		},
		{
			name:     "3 years",
			years:    3,
			expected: types.TermInYearsThreeYears,
		},
		{
			name:     "invalid years defaults to 3 years",
			years:    2,
			expected: types.TermInYearsThreeYears,
		},
		{
			name:     "zero years defaults to 3 years",
			years:    0,
			expected: types.TermInYearsThreeYears,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTermInYears(tt.years)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertLookbackPeriod(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		expected types.LookbackPeriodInDays
	}{
		{
			name:     "7 days",
			days:     7,
			expected: types.LookbackPeriodInDaysSevenDays,
		},
		{
			name:     "30 days",
			days:     30,
			expected: types.LookbackPeriodInDaysThirtyDays,
		},
		{
			name:     "60 days",
			days:     60,
			expected: types.LookbackPeriodInDaysSixtyDays,
		},
		{
			name:     "invalid days defaults to 7 days",
			days:     15,
			expected: types.LookbackPeriodInDaysSevenDays,
		},
		{
			name:     "zero days defaults to 7 days",
			days:     0,
			expected: types.LookbackPeriodInDaysSevenDays,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLookbackPeriod(tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration-style tests (these would require AWS credentials in real usage)
func TestDefaultRecommendationParamsConstruction(t *testing.T) {
	params := RecommendationParams{
		Region:             "us-east-1",
		PaymentOption:      "partial-upfront",
		TermInYears:        3,
		LookbackPeriodDays: 7,
		AccountID:          "123456789012",
	}

	// Test that params are properly constructed
	assert.Equal(t, "us-east-1", params.Region)
	assert.Equal(t, "partial-upfront", params.PaymentOption)
	assert.Equal(t, 3, params.TermInYears)
	assert.Equal(t, 7, params.LookbackPeriodDays)
	assert.Equal(t, "123456789012", params.AccountID)
}

func TestClientRegionProperty(t *testing.T) {
	cfg := aws.Config{Region: "eu-central-1"}
	client := NewClient(cfg)

	assert.Equal(t, "eu-central-1", client.region)
}

// Benchmark tests
func BenchmarkConvertPaymentOption(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertPaymentOption("partial-upfront")
	}
}

func BenchmarkConvertTermInYears(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertTermInYears(3)
	}
}

func BenchmarkConvertLookbackPeriod(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertLookbackPeriod(7)
	}
}

// Edge case tests
func TestParseRecommendedQuantityEdgeCases(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	tests := []struct {
		name      string
		quantity  *string
		expected  int32
		expectErr bool
	}{
		{
			name:     "large quantity",
			quantity: aws.String("999"),
			expected: 999,
		},
		{
			name:     "decimal with high precision",
			quantity: aws.String("5.999999"),
			expected: 5,
		},
		{
			name:     "negative quantity",
			quantity: aws.String("-5"),
			expected: -5, // This might be invalid business logic but tests parsing
		},
		{
			name:     "zero quantity",
			quantity: aws.String("0"),
			expected: 0,
		},
		{
			name:     "very large number",
			quantity: aws.String("2147483647"), // Max int32
			expected: 2147483647,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := &types.ReservationPurchaseRecommendationDetail{
				RecommendedNumberOfInstancesToPurchase: tt.quantity,
			}

			result, err := client.parseRecommendedQuantity(details)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExtractInstanceDetailsWithPartialData(t *testing.T) {
	cfg := aws.Config{Region: "us-west-1"}
	client := NewClient(cfg)

	// Test with minimal required data
	details := &types.ReservationPurchaseRecommendationDetail{
		InstanceDetails: &types.InstanceDetails{
			RDSInstanceDetails: &types.RDSInstanceDetails{
				InstanceType:   aws.String("db.t4g.small"),
				DatabaseEngine: aws.String("aurora-mysql"),
				// Missing region and deployment - should use defaults
			},
		},
	}

	instanceType, engine, region, azConfig, err := client.extractInstanceDetails(details)

	require.NoError(t, err)
	assert.Equal(t, "db.t4g.small", instanceType)
	assert.Equal(t, "aurora-mysql", engine)
	assert.Equal(t, "us-west-1", region)   // Should use client's region
	assert.Equal(t, "single-az", azConfig) // Should default to single-az
}

func TestParseCostInformationWithInvalidFormats(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	tests := []struct {
		name              string
		costAmount        *string
		savingsPercentage *string
		expectedCost      float64
		expectedSavings   float64
	}{
		{
			name:              "valid formats",
			costAmount:        aws.String("1500.75"),
			savingsPercentage: aws.String("25.5"),
			expectedCost:      1500.75,
			expectedSavings:   25.5,
		},
		{
			name:              "cost with currency symbol",
			costAmount:        aws.String("$1500.75"),
			savingsPercentage: aws.String("25.5%"),
			expectedCost:      0.0,  // fmt.Sscanf fails on $ at start, returns 0
			expectedSavings:   25.5, // fmt.Sscanf parses 25.5, stops at %
		},
		{
			name:              "empty strings",
			costAmount:        aws.String(""),
			savingsPercentage: aws.String(""),
			expectedCost:      0.0,
			expectedSavings:   0.0,
		},
		{
			name:              "scientific notation",
			costAmount:        aws.String("1.5e3"),
			savingsPercentage: aws.String("2.5e1"),
			expectedCost:      1500.0, // fmt.Sscanf supports scientific notation
			expectedSavings:   25.0,   // fmt.Sscanf supports scientific notation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := types.ReservationPurchaseRecommendation{
				RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
					TotalEstimatedMonthlySavingsAmount:     tt.costAmount,
					TotalEstimatedMonthlySavingsPercentage: tt.savingsPercentage,
				},
			}

			cost, savings, err := client.parseCostInformation(rec)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCost, cost)
			assert.Equal(t, tt.expectedSavings, savings)
		})
	}
}

// Test helper functions and error conditions
func TestParseRecommendationsWithAllInvalidData(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	// All recommendations are invalid
	awsRecs := []types.ReservationPurchaseRecommendation{
		{
			RecommendationDetails: nil, // Invalid
		},
		{
			RecommendationDetails: []types.ReservationPurchaseRecommendationDetail{
				{
					// Missing required fields
				},
			},
		},
	}

	recommendations, err := client.parseRecommendations(awsRecs, "")

	require.NoError(t, err)
	assert.Empty(t, recommendations) // Should return empty slice, not error
}

func TestParseRecommendationWithMissingInstanceDetails(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	awsRec := types.ReservationPurchaseRecommendation{
		RecommendationDetails: []types.ReservationPurchaseRecommendationDetail{
			{
				RecommendedNumberOfInstancesToPurchase: aws.String("1"),
				InstanceDetails:                        &types.InstanceDetails{
					// Missing RDSInstanceDetails
				},
			},
		},
	}

	rec, err := client.parseRecommendation(awsRec, "")

	assert.Error(t, err)
	assert.Nil(t, rec)
	assert.Contains(t, err.Error(), "failed to extract instance details")
}

func TestParseRecommendedQuantity(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	tests := []struct {
		name      string
		quantity  *string
		expected  int32
		expectErr bool
	}{
		{
			name:     "valid integer quantity",
			quantity: aws.String("5"),
			expected: 5,
		},
		{
			name:     "valid float quantity",
			quantity: aws.String("5.0"),
			expected: 5,
		},
		{
			name:     "valid decimal quantity",
			quantity: aws.String("3.7"),
			expected: 3,
		},
		{
			name:      "nil quantity",
			quantity:  nil,
			expectErr: true,
		},
		{
			name:      "invalid quantity string",
			quantity:  aws.String("invalid"),
			expectErr: true,
		},
		{
			name:      "empty quantity string",
			quantity:  aws.String(""),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := &types.ReservationPurchaseRecommendationDetail{
				RecommendedNumberOfInstancesToPurchase: tt.quantity,
			}

			result, err := client.parseRecommendedQuantity(details)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseCostInformation(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	tests := []struct {
		name            string
		recommendation  types.ReservationPurchaseRecommendation
		expectedCost    float64
		expectedSavings float64
	}{
		{
			name: "valid cost information",
			recommendation: types.ReservationPurchaseRecommendation{
				RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
					TotalEstimatedMonthlySavingsAmount:     aws.String("1500.75"),
					TotalEstimatedMonthlySavingsPercentage: aws.String("25.5"),
				},
			},
			expectedCost:    1500.75,
			expectedSavings: 25.5,
		},
		{
			name: "missing cost information",
			recommendation: types.ReservationPurchaseRecommendation{
				RecommendationSummary: nil,
			},
			expectedCost:    0.0,
			expectedSavings: 0.0,
		},
		{
			name: "partial cost information",
			recommendation: types.ReservationPurchaseRecommendation{
				RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
					TotalEstimatedMonthlySavingsAmount: aws.String("800.25"),
					// Missing percentage
				},
			},
			expectedCost:    800.25,
			expectedSavings: 0.0,
		},
		{
			name: "invalid cost format",
			recommendation: types.ReservationPurchaseRecommendation{
				RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
					TotalEstimatedMonthlySavingsAmount:     aws.String("invalid"),
					TotalEstimatedMonthlySavingsPercentage: aws.String("20.0"),
				},
			},
			expectedCost:    0.0, // Should default to 0 on parse error
			expectedSavings: 20.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, savings, err := client.parseCostInformation(tt.recommendation)

			require.NoError(t, err) // This function doesn't return errors currently
			assert.Equal(t, tt.expectedCost, cost)
			assert.Equal(t, tt.expectedSavings, savings)
		})
	}
}

func TestExtractInstanceDetails(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	tests := []struct {
		name             string
		details          *types.ReservationPurchaseRecommendationDetail
		expectedInstance string
		expectedEngine   string
		expectedRegion   string
		expectedAZConfig string
		expectErr        bool
		errMsg           string
	}{
		{
			name: "valid RDS instance details",
			details: &types.ReservationPurchaseRecommendationDetail{
				InstanceDetails: &types.InstanceDetails{
					RDSInstanceDetails: &types.RDSInstanceDetails{
						InstanceType:     aws.String("db.t4g.medium"),
						DatabaseEngine:   aws.String("mysql"),
						Region:           aws.String("us-east-1"),
						DeploymentOption: aws.String("Single-AZ"),
					},
				},
			},
			expectedInstance: "db.t4g.medium",
			expectedEngine:   "mysql",
			expectedRegion:   "us-east-1",
			expectedAZConfig: "single-az",
		},
		{
			name: "multi-AZ deployment",
			details: &types.ReservationPurchaseRecommendationDetail{
				InstanceDetails: &types.InstanceDetails{
					RDSInstanceDetails: &types.RDSInstanceDetails{
						InstanceType:     aws.String("db.r6g.large"),
						DatabaseEngine:   aws.String("postgres"),
						Region:           aws.String("us-west-2"),
						DeploymentOption: aws.String("Multi-AZ"),
					},
				},
			},
			expectedInstance: "db.r6g.large",
			expectedEngine:   "postgres",
			expectedRegion:   "us-west-2",
			expectedAZConfig: "multi-az",
		},
		{
			name: "missing instance details",
			details: &types.ReservationPurchaseRecommendationDetail{
				InstanceDetails: nil,
			},
			expectErr: true,
			errMsg:    "instance type not found",
		},
		{
			name: "missing RDS instance details",
			details: &types.ReservationPurchaseRecommendationDetail{
				InstanceDetails: &types.InstanceDetails{
					RDSInstanceDetails: nil,
				},
			},
			expectErr: true,
			errMsg:    "instance type not found",
		},
		{
			name: "missing instance type",
			details: &types.ReservationPurchaseRecommendationDetail{
				InstanceDetails: &types.InstanceDetails{
					RDSInstanceDetails: &types.RDSInstanceDetails{
						DatabaseEngine: aws.String("mysql"),
						Region:         aws.String("us-east-1"),
					},
				},
			},
			expectErr: true,
			errMsg:    "instance type not found",
		},
		{
			name: "missing engine",
			details: &types.ReservationPurchaseRecommendationDetail{
				InstanceDetails: &types.InstanceDetails{
					RDSInstanceDetails: &types.RDSInstanceDetails{
						InstanceType: aws.String("db.t4g.medium"),
						Region:       aws.String("us-east-1"),
					},
				},
			},
			expectErr: true,
			errMsg:    "engine not found",
		},
		{
			name: "missing region uses client default",
			details: &types.ReservationPurchaseRecommendationDetail{
				InstanceDetails: &types.InstanceDetails{
					RDSInstanceDetails: &types.RDSInstanceDetails{
						InstanceType:     aws.String("db.t4g.medium"),
						DatabaseEngine:   aws.String("mysql"),
						DeploymentOption: aws.String("Single-AZ"),
					},
				},
			},
			expectedInstance: "db.t4g.medium",
			expectedEngine:   "mysql",
			expectedRegion:   "us-east-1", // From client default
			expectedAZConfig: "single-az",
		},
		{
			name: "missing deployment defaults to single-az",
			details: &types.ReservationPurchaseRecommendationDetail{
				InstanceDetails: &types.InstanceDetails{
					RDSInstanceDetails: &types.RDSInstanceDetails{
						InstanceType:   aws.String("db.t4g.medium"),
						DatabaseEngine: aws.String("mysql"),
						Region:         aws.String("us-east-1"),
					},
				},
			},
			expectedInstance: "db.t4g.medium",
			expectedEngine:   "mysql",
			expectedRegion:   "us-east-1",
			expectedAZConfig: "single-az", // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceType, engine, region, azConfig, err := client.extractInstanceDetails(tt.details)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedInstance, instanceType)
				assert.Equal(t, tt.expectedEngine, engine)
				assert.Equal(t, tt.expectedRegion, region)
				assert.Equal(t, tt.expectedAZConfig, azConfig)
			}
		})
	}
}

func TestParseRecommendation(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	validAwsRec := types.ReservationPurchaseRecommendation{
		RecommendationDetails: []types.ReservationPurchaseRecommendationDetail{
			{
				RecommendedNumberOfInstancesToPurchase: aws.String("3"),
				InstanceDetails: &types.InstanceDetails{
					RDSInstanceDetails: &types.RDSInstanceDetails{
						InstanceType:     aws.String("db.t4g.medium"),
						DatabaseEngine:   aws.String("mysql"),
						Region:           aws.String("us-east-1"),
						DeploymentOption: aws.String("Single-AZ"),
					},
				},
			},
		},
		RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
			TotalEstimatedMonthlySavingsAmount:     aws.String("150.50"),
			TotalEstimatedMonthlySavingsPercentage: aws.String("25.0"),
		},
	}

	tests := []struct {
		name         string
		awsRec       types.ReservationPurchaseRecommendation
		targetRegion string
		expectNil    bool
		expectErr    bool
		errMsg       string
	}{
		{
			name:         "valid recommendation",
			awsRec:       validAwsRec,
			targetRegion: "us-east-1",
			expectNil:    false,
		},
		{
			name:         "region filter excludes recommendation",
			awsRec:       validAwsRec,
			targetRegion: "us-west-2",
			expectNil:    true, // Should return nil (filtered out)
		},
		{
			name:         "empty target region accepts all",
			awsRec:       validAwsRec,
			targetRegion: "",
			expectNil:    false,
		},
		{
			name: "missing recommendation details",
			awsRec: types.ReservationPurchaseRecommendation{
				RecommendationDetails: nil,
			},
			targetRegion: "us-east-1",
			expectErr:    true,
			errMsg:       "recommendation details are missing",
		},
		{
			name: "invalid quantity",
			awsRec: types.ReservationPurchaseRecommendation{
				RecommendationDetails: []types.ReservationPurchaseRecommendationDetail{
					{
						RecommendedNumberOfInstancesToPurchase: aws.String("invalid"),
						InstanceDetails: &types.InstanceDetails{
							RDSInstanceDetails: &types.RDSInstanceDetails{
								InstanceType:   aws.String("db.t4g.medium"),
								DatabaseEngine: aws.String("mysql"),
								Region:         aws.String("us-east-1"),
							},
						},
					},
				},
			},
			targetRegion: "us-east-1",
			expectErr:    true,
			errMsg:       "failed to parse recommended quantity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, err := client.parseRecommendation(tt.awsRec, tt.targetRegion)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, rec)
			} else if tt.expectNil {
				require.NoError(t, err)
				assert.Nil(t, rec)
			} else {
				require.NoError(t, err)
				require.NotNil(t, rec)

				// Verify the parsed recommendation
				assert.Equal(t, "us-east-1", rec.Region)
				assert.Equal(t, "db.t4g.medium", rec.InstanceType)
				assert.Equal(t, "mysql", rec.Engine)
				assert.Equal(t, "single-az", rec.AZConfig)
				assert.Equal(t, "partial-upfront", rec.PaymentOption)
				assert.Equal(t, int32(36), rec.Term)
				assert.Equal(t, int32(3), rec.Count)
				assert.Equal(t, 150.50, rec.EstimatedCost)
				assert.Equal(t, 25.0, rec.SavingsPercent)
				assert.NotEmpty(t, rec.Description)
			}
		})
	}
}

func TestParseRecommendations(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	client := NewClient(cfg)

	awsRecs := []types.ReservationPurchaseRecommendation{
		{
			RecommendationDetails: []types.ReservationPurchaseRecommendationDetail{
				{
					RecommendedNumberOfInstancesToPurchase: aws.String("2"),
					InstanceDetails: &types.InstanceDetails{
						RDSInstanceDetails: &types.RDSInstanceDetails{
							InstanceType:     aws.String("db.t4g.medium"),
							DatabaseEngine:   aws.String("mysql"),
							Region:           aws.String("us-east-1"),
							DeploymentOption: aws.String("Single-AZ"),
						},
					},
				},
			},
			RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
				TotalEstimatedMonthlySavingsAmount:     aws.String("100.00"),
				TotalEstimatedMonthlySavingsPercentage: aws.String("20.0"),
			},
		},
		{
			RecommendationDetails: []types.ReservationPurchaseRecommendationDetail{
				{
					RecommendedNumberOfInstancesToPurchase: aws.String("1"),
					InstanceDetails: &types.InstanceDetails{
						RDSInstanceDetails: &types.RDSInstanceDetails{
							InstanceType:     aws.String("db.r6g.large"),
							DatabaseEngine:   aws.String("postgres"),
							Region:           aws.String("us-west-2"),
							DeploymentOption: aws.String("Multi-AZ"),
						},
					},
				},
			},
			RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
				TotalEstimatedMonthlySavingsAmount:     aws.String("200.00"),
				TotalEstimatedMonthlySavingsPercentage: aws.String("30.0"),
			},
		},
		{
			RecommendationDetails: []types.ReservationPurchaseRecommendationDetail{
				{
					RecommendedNumberOfInstancesToPurchase: aws.String("3"),
					InstanceDetails: &types.InstanceDetails{
						RDSInstanceDetails: &types.RDSInstanceDetails{
							InstanceType:     aws.String("db.t4g.small"),
							DatabaseEngine:   aws.String("aurora-mysql"),
							Region:           aws.String("eu-central-1"),
							DeploymentOption: aws.String("Single-AZ"),
						},
					},
				},
			},
			RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
				TotalEstimatedMonthlySavingsAmount:     aws.String("150.00"),
				TotalEstimatedMonthlySavingsPercentage: aws.String("25.0"),
			},
		},
		{
			// Invalid recommendation (missing details)
			RecommendationDetails: nil,
		},
		{
			// Another valid recommendation for us-east-1
			RecommendationDetails: []types.ReservationPurchaseRecommendationDetail{
				{
					RecommendedNumberOfInstancesToPurchase: aws.String("5"),
					InstanceDetails: &types.InstanceDetails{
						RDSInstanceDetails: &types.RDSInstanceDetails{
							InstanceType:     aws.String("db.r6g.xlarge"),
							DatabaseEngine:   aws.String("aurora-postgresql"),
							Region:           aws.String("us-east-1"),
							DeploymentOption: aws.String("Multi-AZ"),
						},
					},
				},
			},
			RecommendationSummary: &types.ReservationPurchaseRecommendationSummary{
				TotalEstimatedMonthlySavingsAmount:     aws.String("500.00"),
				TotalEstimatedMonthlySavingsPercentage: aws.String("35.0"),
			},
		},
	}

	tests := []struct {
		name            string
		targetRegion    string
		expectedCount   int
		expectedEngines []string
		expectedRegions []string
	}{
		{
			name:            "no region filter",
			targetRegion:    "",
			expectedCount:   4, // Should get 4 valid recommendations (1 invalid skipped)
			expectedEngines: []string{"mysql", "postgres", "aurora-mysql", "aurora-postgresql"},
			expectedRegions: []string{"us-east-1", "us-west-2", "eu-central-1", "us-east-1"},
		},
		{
			name:            "filter by us-east-1",
			targetRegion:    "us-east-1",
			expectedCount:   2, // MySQL and Aurora PostgreSQL recommendations
			expectedEngines: []string{"mysql", "aurora-postgresql"},
			expectedRegions: []string{"us-east-1", "us-east-1"},
		},
		{
			name:            "filter by us-west-2",
			targetRegion:    "us-west-2",
			expectedCount:   1, // Only PostgreSQL recommendation
			expectedEngines: []string{"postgres"},
			expectedRegions: []string{"us-west-2"},
		},
		{
			name:            "filter by eu-central-1",
			targetRegion:    "eu-central-1",
			expectedCount:   1, // Only Aurora MySQL recommendation
			expectedEngines: []string{"aurora-mysql"},
			expectedRegions: []string{"eu-central-1"},
		},
		{
			name:            "filter by non-existent region",
			targetRegion:    "ap-southeast-1",
			expectedCount:   0, // No recommendations
			expectedEngines: []string{},
			expectedRegions: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations, err := client.parseRecommendations(awsRecs, tt.targetRegion)

			require.NoError(t, err)
			assert.Len(t, recommendations, tt.expectedCount)

			// Verify engines match expected
			engines := make([]string, len(recommendations))
			regions := make([]string, len(recommendations))
			for i, rec := range recommendations {
				engines[i] = rec.Engine
				regions[i] = rec.Region
			}

			assert.ElementsMatch(t, tt.expectedEngines, engines)
			assert.ElementsMatch(t, tt.expectedRegions, regions)

			// Verify all recommendations have required fields
			for _, rec := range recommendations {
				assert.NotEmpty(t, rec.Region)
				assert.NotEmpty(t, rec.Engine)
				assert.NotEmpty(t, rec.InstanceType)
				assert.NotEmpty(t, rec.AZConfig)
				assert.Greater(t, rec.Count, int32(0))
				assert.GreaterOrEqual(t, rec.EstimatedCost, 0.0)
				assert.GreaterOrEqual(t, rec.SavingsPercent, 0.0)
				assert.NotEmpty(t, rec.Description)
			}
		})
	}
}

func TestNormalizeRegionName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "US East (N. Virginia) to us-east-1",
			input:    "US East (N. Virginia)",
			expected: "us-east-1",
		},
		{
			name:     "US East (Ohio) to us-east-2",
			input:    "US East (Ohio)",
			expected: "us-east-2",
		},
		{
			name:     "US West (Oregon) to us-west-2",
			input:    "US West (Oregon)",
			expected: "us-west-2",
		},
		{
			name:     "Europe (Ireland) to eu-west-1",
			input:    "Europe (Ireland)",
			expected: "eu-west-1",
		},
		{
			name:     "Europe (Frankfurt) to eu-central-1",
			input:    "Europe (Frankfurt)",
			expected: "eu-central-1",
		},
		{
			name:     "Asia Pacific (Tokyo) to ap-northeast-1",
			input:    "Asia Pacific (Tokyo)",
			expected: "ap-northeast-1",
		},
		{
			name:     "Asia Pacific (Singapore) to ap-southeast-1",
			input:    "Asia Pacific (Singapore)",
			expected: "ap-southeast-1",
		},
		{
			name:     "Already valid region code",
			input:    "us-east-1",
			expected: "us-east-1",
		},
		{
			name:     "Already valid region code eu-west-1",
			input:    "eu-west-1",
			expected: "eu-west-1",
		},
		{
			name:     "Case insensitive matching",
			input:    "us east (n. virginia)",
			expected: "us-east-1",
		},
		{
			name:     "Partial matching - virginia",
			input:    "virginia",
			expected: "us-east-1",
		},
		{
			name:     "Partial matching - ohio",
			input:    "ohio",
			expected: "us-east-2",
		},
		{
			name:     "Partial matching - oregon",
			input:    "oregon",
			expected: "us-west-2",
		},
		{
			name:     "Partial matching - ireland",
			input:    "ireland",
			expected: "eu-west-1",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Unknown region returns original",
			input:    "Mars (Red Planet)",
			expected: "Mars (Red Planet)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRegionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRegionCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid region code us-east-1",
			input:    "us-east-1",
			expected: true,
		},
		{
			name:     "Valid region code eu-central-1",
			input:    "eu-central-1",
			expected: true,
		},
		{
			name:     "Valid region code ap-southeast-2",
			input:    "ap-southeast-2",
			expected: true,
		},
		{
			name:     "Human readable region name",
			input:    "US East (N. Virginia)",
			expected: false,
		},
		{
			name:     "Mixed case",
			input:    "US-EAST-1",
			expected: false,
		},
		{
			name:     "No dashes",
			input:    "useast1",
			expected: false,
		},
		{
			name:     "Contains spaces",
			input:    "us east 1",
			expected: false,
		},
		{
			name:     "Contains parentheses",
			input:    "us-east-(1)",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRegionCode(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkNormalizeRegionName(b *testing.B) {
	testInputs := []string{
		"US East (N. Virginia)",
		"us-east-1",
		"Europe (Frankfurt)",
		"unknown region",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range testInputs {
			normalizeRegionName(input)
		}
	}
}
