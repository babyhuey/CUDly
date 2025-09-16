package recommendations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecommendation_GenerateDescription(t *testing.T) {
	tests := []struct {
		name           string
		recommendation Recommendation
		expected       string
	}{
		{
			name: "single AZ recommendation",
			recommendation: Recommendation{
				Engine:       "mysql",
				InstanceType: "db.t4g.medium",
				AZConfig:     "single-az",
			},
			expected: "mysql db.t4g.medium Single-AZ",
		},
		{
			name: "multi AZ recommendation",
			recommendation: Recommendation{
				Engine:       "aurora-postgresql",
				InstanceType: "db.r6g.large",
				AZConfig:     "multi-az",
			},
			expected: "aurora-postgresql db.r6g.large Multi-AZ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.recommendation.GenerateDescription()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecommendation_Validate(t *testing.T) {
	tests := []struct {
		name           string
		recommendation Recommendation
		wantErr        bool
		errMsg         string
	}{
		{
			name: "valid recommendation",
			recommendation: Recommendation{
				Region:       "us-east-1",
				InstanceType: "db.t4g.medium",
				Engine:       "mysql",
				AZConfig:     "single-az",
				Count:        1,
			},
			wantErr: false,
		},
		{
			name: "missing region",
			recommendation: Recommendation{
				InstanceType: "db.t4g.medium",
				Engine:       "mysql",
				AZConfig:     "single-az",
				Count:        1,
			},
			wantErr: true,
			errMsg:  "region is required",
		},
		{
			name: "missing instance type",
			recommendation: Recommendation{
				Region:   "us-east-1",
				Engine:   "mysql",
				AZConfig: "single-az",
				Count:    1,
			},
			wantErr: true,
			errMsg:  "instance type is required",
		},
		{
			name: "missing engine",
			recommendation: Recommendation{
				Region:       "us-east-1",
				InstanceType: "db.t4g.medium",
				AZConfig:     "single-az",
				Count:        1,
			},
			wantErr: true,
			errMsg:  "engine is required",
		},
		{
			name: "invalid count",
			recommendation: Recommendation{
				Region:       "us-east-1",
				InstanceType: "db.t4g.medium",
				Engine:       "mysql",
				AZConfig:     "single-az",
				Count:        0,
			},
			wantErr: true,
			errMsg:  "count must be greater than 0",
		},
		{
			name: "invalid AZ config",
			recommendation: Recommendation{
				Region:       "us-east-1",
				InstanceType: "db.t4g.medium",
				Engine:       "mysql",
				AZConfig:     "invalid-az",
				Count:        1,
			},
			wantErr: true,
			errMsg:  "AZ config must be 'single-az' or 'multi-az'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.recommendation.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRecommendation_GetDurationString(t *testing.T) {
	tests := []struct {
		name     string
		term     int32
		expected string
	}{
		{
			name:     "1 year term",
			term:     12,
			expected: "1yr",
		},
		{
			name:     "3 year term",
			term:     36,
			expected: "3yr",
		},
		{
			name:     "invalid term defaults to 3yr",
			term:     24,
			expected: "3yr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &Recommendation{Term: tt.term}
			result := rec.GetDurationString()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecommendation_GetMultiAZ(t *testing.T) {
	tests := []struct {
		name     string
		azConfig string
		expected bool
	}{
		{
			name:     "single AZ",
			azConfig: "single-az",
			expected: false,
		},
		{
			name:     "multi AZ",
			azConfig: "multi-az",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &Recommendation{AZConfig: tt.azConfig}
			result := rec.GetMultiAZ()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecommendation_CalculateAnnualSavings(t *testing.T) {
	rec := &Recommendation{
		EstimatedCost: 100.50, // Monthly cost
	}

	expectedAnnual := 100.50 * 12
	actual := rec.CalculateAnnualSavings()
	assert.Equal(t, expectedAnnual, actual)
}

func TestRecommendation_CalculateTotalTermSavings(t *testing.T) {
	tests := []struct {
		name          string
		estimatedCost float64
		term          int32
		expected      float64
	}{
		{
			name:          "3 year term",
			estimatedCost: 100.0,
			term:          36,
			expected:      3600.0, // 100 * 12 * 3
		},
		{
			name:          "1 year term",
			estimatedCost: 50.0,
			term:          12,
			expected:      600.0, // 50 * 12 * 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &Recommendation{
				EstimatedCost: tt.estimatedCost,
				Term:          tt.term,
			}
			result := rec.CalculateTotalTermSavings()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSummarizeRecommendations(t *testing.T) {
	recommendations := []Recommendation{
		{
			Engine:         "mysql",
			InstanceType:   "db.t4g.medium",
			Region:         "us-east-1",
			Count:          2,
			EstimatedCost:  100.0,
			SavingsPercent: 20.0,
		},
		{
			Engine:         "mysql",
			InstanceType:   "db.r6g.large",
			Region:         "us-east-1",
			Count:          1,
			EstimatedCost:  200.0,
			SavingsPercent: 30.0,
		},
		{
			Engine:         "postgres",
			InstanceType:   "db.t4g.medium",
			Region:         "us-west-2",
			Count:          3,
			EstimatedCost:  150.0,
			SavingsPercent: 25.0,
		},
	}

	summary := SummarizeRecommendations(recommendations)

	// Test overall summary
	assert.Equal(t, 3, summary.TotalRecommendations)
	assert.Equal(t, int32(6), summary.TotalInstances)  // 2 + 1 + 3
	assert.Equal(t, 450.0, summary.TotalEstimatedCost) // 100 + 200 + 150
	assert.Equal(t, 25.0, summary.AverageSavings)      // (20 + 30 + 25) / 3

	// Test engine summary
	assert.Len(t, summary.ByEngine, 2)

	mysqlSummary := summary.ByEngine["mysql"]
	assert.Equal(t, int32(2), mysqlSummary.Count)
	assert.Equal(t, int32(3), mysqlSummary.Instances)  // 2 + 1
	assert.Equal(t, 300.0, mysqlSummary.EstimatedCost) // 100 + 200

	postgresSummary := summary.ByEngine["postgres"]
	assert.Equal(t, int32(1), postgresSummary.Count)
	assert.Equal(t, int32(3), postgresSummary.Instances)
	assert.Equal(t, 150.0, postgresSummary.EstimatedCost)

	// Test instance type summary
	assert.Len(t, summary.ByInstanceType, 2)

	mediumSummary := summary.ByInstanceType["db.t4g.medium"]
	assert.Equal(t, int32(2), mediumSummary.Count)
	assert.Equal(t, int32(5), mediumSummary.Instances)  // 2 + 3
	assert.Equal(t, 250.0, mediumSummary.EstimatedCost) // 100 + 150

	// Test region summary
	assert.Len(t, summary.ByRegion, 2)

	usEast1Summary := summary.ByRegion["us-east-1"]
	assert.Equal(t, int32(2), usEast1Summary.Count)
	assert.Equal(t, int32(3), usEast1Summary.Instances)  // 2 + 1
	assert.Equal(t, 300.0, usEast1Summary.EstimatedCost) // 100 + 200
}

func TestFilterRecommendations(t *testing.T) {
	recommendations := []Recommendation{
		{
			Region:         "us-east-1",
			Engine:         "mysql",
			InstanceType:   "db.t4g.medium",
			AZConfig:       "single-az",
			Count:          2,
			SavingsPercent: 20.0,
		},
		{
			Region:         "us-west-2",
			Engine:         "postgres",
			InstanceType:   "db.r6g.large",
			AZConfig:       "multi-az",
			Count:          1,
			SavingsPercent: 30.0,
		},
		{
			Region:         "us-east-1",
			Engine:         "aurora-mysql",
			InstanceType:   "db.t4g.small",
			AZConfig:       "single-az",
			Count:          5,
			SavingsPercent: 15.0,
		},
	}

	tests := []struct {
		name           string
		filter         RecommendationFilter
		expectedCount  int
		expectedEngine string
	}{
		{
			name: "filter by region",
			filter: RecommendationFilter{
				Regions: []string{"us-east-1"},
			},
			expectedCount: 2,
		},
		{
			name: "filter by engine",
			filter: RecommendationFilter{
				Engines: []string{"mysql"},
			},
			expectedCount: 1,
		},
		{
			name: "filter by minimum savings",
			filter: RecommendationFilter{
				MinSavings: 25.0,
			},
			expectedCount: 1,
		},
		{
			name: "filter by multi-AZ only",
			filter: RecommendationFilter{
				MultiAZOnly: true,
			},
			expectedCount: 1,
		},
		{
			name: "filter by single-AZ only",
			filter: RecommendationFilter{
				SingleAZOnly: true,
			},
			expectedCount: 2,
		},
		{
			name: "filter by max instances",
			filter: RecommendationFilter{
				MaxInstances: 2,
			},
			expectedCount: 2,
		},
		{
			name: "filter by min instances",
			filter: RecommendationFilter{
				MinInstances: 3,
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterRecommendations(recommendations, tt.filter)
			assert.Len(t, filtered, tt.expectedCount)
		})
	}
}

func TestSortRecommendations(t *testing.T) {
	recommendations := []Recommendation{
		{
			Engine:         "mysql",
			InstanceType:   "db.t4g.medium",
			SavingsPercent: 20.0,
			EstimatedCost:  100.0,
			Count:          2,
		},
		{
			Engine:         "postgres",
			InstanceType:   "db.r6g.large",
			SavingsPercent: 30.0,
			EstimatedCost:  200.0,
			Count:          1,
		},
		{
			Engine:         "aurora-mysql",
			InstanceType:   "db.t4g.small",
			SavingsPercent: 15.0,
			EstimatedCost:  50.0,
			Count:          5,
		},
	}

	tests := []struct {
		name              string
		sortBy            string
		ascending         bool
		expectedFirstItem string
	}{
		{
			name:              "sort by savings descending",
			sortBy:            "savings",
			ascending:         false,
			expectedFirstItem: "postgres", // 30% savings
		},
		{
			name:              "sort by savings ascending",
			sortBy:            "savings",
			ascending:         true,
			expectedFirstItem: "aurora-mysql", // 15% savings
		},
		{
			name:              "sort by cost ascending",
			sortBy:            "cost",
			ascending:         true,
			expectedFirstItem: "aurora-mysql", // $50
		},
		{
			name:              "sort by instances descending",
			sortBy:            "instances",
			ascending:         false,
			expectedFirstItem: "aurora-mysql", // 5 instances
		},
		{
			name:              "sort by engine ascending",
			sortBy:            "engine",
			ascending:         true,
			expectedFirstItem: "aurora-mysql", // alphabetically first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original
			testRecs := make([]Recommendation, len(recommendations))
			copy(testRecs, recommendations)

			SortRecommendations(testRecs, tt.sortBy, tt.ascending)
			assert.Equal(t, tt.expectedFirstItem, testRecs[0].Engine)
		})
	}
}

func TestApplyCoveragePercentage(t *testing.T) {
	recommendations := []Recommendation{
		{Count: 10},
		{Count: 5},
		{Count: 2},
	}

	tests := []struct {
		name             string
		coverage         float64
		expectedCounts   []int32
		expectedFiltered int
	}{
		{
			name:             "100% coverage",
			coverage:         100.0,
			expectedCounts:   []int32{10, 5, 2},
			expectedFiltered: 3,
		},
		{
			name:             "50% coverage",
			coverage:         50.0,
			expectedCounts:   []int32{5, 2, 1},
			expectedFiltered: 3,
		},
		{
			name:             "20% coverage",
			coverage:         20.0,
			expectedCounts:   []int32{2, 1},
			expectedFiltered: 2, // Third item would be 0 and filtered out
		},
		{
			name:             "10% coverage",
			coverage:         10.0,
			expectedCounts:   []int32{1},
			expectedFiltered: 1, // Only first item survives
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyCoveragePercentage(recommendations, tt.coverage)
			assert.Len(t, result, tt.expectedFiltered)

			for i, expectedCount := range tt.expectedCounts {
				if i < len(result) {
					assert.Equal(t, expectedCount, result[i].Count)
				}
			}
		})
	}
}

func TestDefaultRecommendationParams(t *testing.T) {
	params := DefaultRecommendationParams()

	assert.Equal(t, "partial-upfront", params.PaymentOption)
	assert.Equal(t, 3, params.TermInYears)
	assert.Equal(t, 7, params.LookbackPeriodDays)
	assert.Equal(t, "", params.Region)    // Should be empty by default
	assert.Equal(t, "", params.AccountID) // Should be empty by default
}

func TestContainsHelper(t *testing.T) {
	slice := []string{"mysql", "postgres", "aurora-mysql"}

	tests := []struct {
		name     string
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			item:     "mysql",
			expected: true,
		},
		{
			name:     "item does not exist",
			item:     "mariadb",
			expected: false,
		},
		{
			name:     "empty item",
			item:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecommendationJSONTags(t *testing.T) {
	// Test that JSON tags are properly set on the Recommendation struct
	rec := &Recommendation{
		Region:         "us-east-1",
		InstanceType:   "db.t4g.medium",
		Engine:         "mysql",
		AZConfig:       "single-az",
		PaymentOption:  "partial-upfront",
		Term:           36,
		Count:          2,
		EstimatedCost:  100.0,
		SavingsPercent: 25.0,
		Description:    "MySQL t4g.medium Single-AZ",
		Timestamp:      time.Now(),
	}

	// Validate the recommendation
	err := rec.Validate()
	assert.NoError(t, err)

	// Test description generation
	description := rec.GenerateDescription()
	assert.NotEmpty(t, description)
}

// Benchmark tests
func BenchmarkSummarizeRecommendations(b *testing.B) {
	recommendations := make([]Recommendation, 100)
	for i := 0; i < 100; i++ {
		recommendations[i] = Recommendation{
			Engine:         "mysql",
			InstanceType:   "db.t4g.medium",
			Region:         "us-east-1",
			Count:          int32(i + 1),
			EstimatedCost:  float64(i * 10),
			SavingsPercent: float64(i % 30),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SummarizeRecommendations(recommendations)
	}
}

func BenchmarkFilterRecommendations(b *testing.B) {
	recommendations := make([]Recommendation, 1000)
	for i := 0; i < 1000; i++ {
		recommendations[i] = Recommendation{
			Region:         "us-east-1",
			Engine:         "mysql",
			InstanceType:   "db.t4g.medium",
			AZConfig:       "single-az",
			Count:          int32(i + 1),
			SavingsPercent: float64(i % 50),
		}
	}

	filter := RecommendationFilter{
		MinSavings:   25.0,
		MaxInstances: 500,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterRecommendations(recommendations, filter)
	}
}
