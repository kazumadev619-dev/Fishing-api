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

func weatherOKBody() map[string]interface{} {
	return map[string]interface{}{
		"main":    map[string]interface{}{"temp": 20.5, "feels_like": 19.0, "pressure": 1013.0, "humidity": 65},
		"wind":    map[string]interface{}{"speed": 3.5, "deg": 180},
		"weather": []map[string]interface{}{{"description": "晴れ"}},
		"dt":      1700000000,
	}
}

func TestWeatherClient_FetchCurrent(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		wantTemp   float64
	}{
		{name: "正常取得", statusCode: http.StatusOK, wantErr: false, wantTemp: 20.5},
		{name: "APIエラー500", statusCode: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/data/2.5/weather", r.URL.Path)
				assert.Equal(t, "35.689500", r.URL.Query().Get("lat"))
				assert.Equal(t, "139.691700", r.URL.Query().Get("lon"))
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(weatherOKBody())
				}
			}))
			defer server.Close()

			client := newWeatherClientWithHTTPClient("test-api-key", server.URL, &http.Client{})
			result, err := client.FetchCurrent(context.Background(), 35.6895, 139.6917)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTemp, result.Temperature)
				assert.Equal(t, 3.5, result.WindSpeed)
				assert.Equal(t, "晴れ", result.Description)
			}
		})
	}
}

func TestWeatherClient_FetchForecast(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
		wantLen    int
	}{
		{name: "正常取得", statusCode: http.StatusOK, wantErr: false, wantLen: 1},
		{name: "APIエラー500", statusCode: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/data/2.5/forecast", r.URL.Path)
				assert.Equal(t, "35.689500", r.URL.Query().Get("lat"))
				assert.Equal(t, "139.691700", r.URL.Query().Get("lon"))
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"list": []map[string]interface{}{
							{
								"main":    map[string]interface{}{"temp": 18.0, "feels_like": 17.0, "pressure": 1010.0, "humidity": 70},
								"wind":    map[string]interface{}{"speed": 2.0, "deg": 90},
								"weather": []map[string]interface{}{{"description": "曇り"}},
								"dt":      1700003600,
							},
						},
					})
				}
			}))
			defer server.Close()

			client := newWeatherClientWithHTTPClient("test-api-key", server.URL, &http.Client{})
			results, err := client.FetchForecast(context.Background(), 35.6895, 139.6917)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, results)
			} else {
				require.NoError(t, err)
				assert.Len(t, results, tt.wantLen)
				assert.Equal(t, 18.0, results[0].Temperature)
			}
		})
	}
}
