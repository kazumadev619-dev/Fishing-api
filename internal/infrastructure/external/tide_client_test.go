package external

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tideOKBody() map[string]interface{} {
	return map[string]interface{}{
		"tide_type": "大潮",
		"high_tides": []map[string]interface{}{
			{"time": "06:15", "height": 180.5},
			{"time": "18:30", "height": 175.0},
		},
		"low_tides": []map[string]interface{}{
			{"time": "12:45", "height": 30.2},
		},
	}
}

func TestTideClient_FetchTideData(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		wantErr      bool
		wantTideType string
		wantHighLen  int
		wantLowLen   int
		wantHeight   float64
	}{
		{
			name:         "正常取得",
			statusCode:   http.StatusOK,
			wantErr:      false,
			wantTideType: "大潮",
			wantHighLen:  2,
			wantLowLen:   1,
			wantHeight:   180.5,
		},
		{
			name:       "APIエラー500",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/get_tide/14/1341T/2026-04-30/", r.URL.Path)
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tideOKBody()) //nolint:errcheck
				}
			}))
			defer server.Close()

			client := newTideClientWithHTTPClient(server.URL, &http.Client{})
			result, err := client.FetchTideData(context.Background(), "14", "1341T", "2026-04-30")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTideType, result.TideType)
				assert.Equal(t, "1341T", result.PortCode)
				assert.Equal(t, "2026-04-30", result.Date)
				assert.Len(t, result.HighTides, tt.wantHighLen)
				assert.Len(t, result.LowTides, tt.wantLowLen)
				assert.Equal(t, tt.wantHeight, result.HighTides[0].Height)
				assert.Equal(t, 6, result.HighTides[0].Time.Hour())
				assert.Equal(t, 15, result.HighTides[0].Time.Minute())
				assert.Equal(t, 30.2, result.LowTides[0].Height)
			}
		})
	}
}
