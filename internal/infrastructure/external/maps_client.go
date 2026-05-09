package external

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/kazumadev619-dev/fishing-api/internal/domain/entity"
)

type MapsClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewMapsClient(apiKey string) *MapsClient {
	return &MapsClient{
		apiKey:  apiKey,
		baseURL: "https://maps.googleapis.com",
		client:  newHTTPClient(),
	}
}

func newMapsClientWithHTTPClient(apiKey, baseURL string, httpClient *http.Client) *MapsClient {
	return &MapsClient{apiKey: apiKey, baseURL: baseURL, client: httpClient}
}

type geocodeResponse struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
		AddressComponents []struct {
			LongName string   `json:"long_name"`
			Types    []string `json:"types"`
		} `json:"address_components"`
	} `json:"results"`
	Status string `json:"status"`
}

func (c *MapsClient) SearchLocations(ctx context.Context, query string, limit int) ([]*entity.LocationSearchResult, error) {
	params := url.Values{}
	params.Set("address", query)
	params.Set("key", c.apiKey)
	params.Set("language", "ja")
	params.Set("region", "jp")

	apiURL := fmt.Sprintf("%s/maps/api/geocode/json?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating maps request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("maps API request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Warn("failed to close maps response body", "error", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("maps API returned status %d", resp.StatusCode)
	}

	var data geocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode maps response: %w", err)
	}

	if data.Status != "OK" && data.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("maps API error: %s", data.Status)
	}

	var results []*entity.LocationSearchResult
	for i, r := range data.Results {
		if i >= limit {
			break
		}
		result := &entity.LocationSearchResult{
			Name:      r.FormattedAddress,
			Latitude:  r.Geometry.Location.Lat,
			Longitude: r.Geometry.Location.Lng,
		}
		for _, comp := range r.AddressComponents {
			for _, t := range comp.Types {
				if t == "administrative_area_level_1" {
					result.Prefecture = comp.LongName
				}
			}
		}
		results = append(results, result)
	}
	return results, nil
}
