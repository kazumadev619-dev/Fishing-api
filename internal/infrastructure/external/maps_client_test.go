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

func geocodeOKBody() map[string]interface{} {
	return map[string]interface{}{
		"status": "OK",
		"results": []map[string]interface{}{
			{
				"formatted_address": "静岡県静岡市",
				"geometry": map[string]interface{}{
					"location": map[string]interface{}{
						"lat": 34.9769,
						"lng": 138.3831,
					},
				},
				"address_components": []map[string]interface{}{
					{
						"long_name": "静岡県",
						"types":     []string{"administrative_area_level_1", "political"},
					},
					{
						"long_name": "静岡市",
						"types":     []string{"locality", "political"},
					},
				},
			},
		},
	}
}

func TestMapsClient_SearchLocations(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   map[string]interface{}
		limit          int
		wantErr        bool
		wantLen        int
		wantName       string
		wantLat        float64
		wantLon        float64
		wantPrefecture string
	}{
		{
			name:           "正常取得",
			statusCode:     http.StatusOK,
			responseBody:   geocodeOKBody(),
			limit:          5,
			wantErr:        false,
			wantLen:        1,
			wantName:       "静岡県静岡市",
			wantLat:        34.9769,
			wantLon:        138.3831,
			wantPrefecture: "静岡県",
		},
		{
			name:       "ZERO_RESULTS",
			statusCode: http.StatusOK,
			responseBody: map[string]interface{}{
				"status":  "ZERO_RESULTS",
				"results": []map[string]interface{}{},
			},
			limit:   5,
			wantErr: false,
			wantLen: 0,
		},
		{
			name:       "APIエラー(REQUEST_DENIED)",
			statusCode: http.StatusOK,
			responseBody: map[string]interface{}{
				"status":  "REQUEST_DENIED",
				"results": []map[string]interface{}{},
			},
			limit:   5,
			wantErr: true,
		},
		{
			name:         "limit制限",
			statusCode:   http.StatusOK,
			responseBody: geocodeOKBody(),
			limit:        0,
			wantErr:      false,
			wantLen:      0,
		},
		{
			name:       "HTTP503エラー",
			statusCode: http.StatusServiceUnavailable,
			limit:      5,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/maps/api/geocode/json", r.URL.Path)
				assert.NotEmpty(t, r.URL.Query().Get("address"))
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != nil {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			client := newMapsClientWithHTTPClient("test-key", server.URL, &http.Client{})
			results, err := client.SearchLocations(context.Background(), "静岡", tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, results)
			} else {
				require.NoError(t, err)
				assert.Len(t, results, tt.wantLen)
				if tt.wantLen > 0 {
					assert.Equal(t, tt.wantName, results[0].Name)
					assert.Equal(t, tt.wantLat, results[0].Latitude)
					assert.Equal(t, tt.wantLon, results[0].Longitude)
					assert.Equal(t, tt.wantPrefecture, results[0].Prefecture)
				}
			}
		})
	}
}
