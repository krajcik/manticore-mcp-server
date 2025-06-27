package mcp

import (
	"encoding/json"
	"testing"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_successResponse(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		response *Response
		wantErr  bool
	}{
		{
			name: "basic success response",
			response: &Response{
				Success: true,
				Data:    map[string]interface{}{"test": "data"},
				Meta: &Meta{
					Total:     10,
					Count:     5,
					Operation: "search",
				},
			},
			wantErr: false,
		},
		{
			name: "success response with string data",
			response: &Response{
				Success: true,
				Data:    "simple string data",
				Meta: &Meta{
					Operation: "status",
				},
			},
			wantErr: false,
		},
		{
			name: "success response with array data",
			response: &Response{
				Success: true,
				Data:    []interface{}{"item1", "item2", "item3"},
				Meta: &Meta{
					Total:     3,
					Count:     3,
					Operation: "list",
				},
			},
			wantErr: false,
		},
		{
			name: "success response with nil data",
			response: &Response{
				Success: true,
				Data:    nil,
				Meta: &Meta{
					Operation: "delete",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.successResponse(tt.response)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Verify the response is valid MCP ToolResponse
				assert.IsType(t, &mcp_golang.ToolResponse{}, result)

				// Verify the content can be parsed as JSON
				content := result.Content[0].TextContent.Text
				var parsedResponse Response
				err := json.Unmarshal([]byte(content), &parsedResponse)
				require.NoError(t, err)

				// Verify the parsed response matches expected
				assert.Equal(t, tt.response.Success, parsedResponse.Success)
				assert.Equal(t, tt.response.Data, parsedResponse.Data)
				assert.Equal(t, tt.response.Meta, parsedResponse.Meta)
			}
		})
	}
}

func TestRegistry_errorResponse(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "simple error message",
			message: "Something went wrong",
		},
		{
			name:    "detailed error message",
			message: "Failed to connect to Manticore Search: connection refused",
		},
		{
			name:    "empty error message",
			message: "",
		},
		{
			name:    "error with special characters",
			message: "Error: table 'test' not found (errno: 1146)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.errorResponse(tt.message)

			require.NoError(t, err)
			require.NotNil(t, result)

			// Verify the response is valid MCP ToolResponse
			assert.IsType(t, &mcp_golang.ToolResponse{}, result)

			// Verify the content can be parsed as JSON
			content := result.Content[0].TextContent.Text
			var parsedResponse Response
			err = json.Unmarshal([]byte(content), &parsedResponse)
			require.NoError(t, err)

			// Verify error response structure
			assert.False(t, parsedResponse.Success)
			assert.Equal(t, tt.message, parsedResponse.Error)
			assert.Nil(t, parsedResponse.Data)
			assert.Nil(t, parsedResponse.Meta)
		})
	}
}

func TestResponse_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
	}{
		{
			name: "complete response",
			response: &Response{
				Success: true,
				Data: map[string]interface{}{
					"results": []interface{}{
						map[string]interface{}{"id": float64(1), "title": "Test Document"},
						map[string]interface{}{"id": float64(2), "title": "Another Document"},
					},
				},
				Meta: &Meta{
					Total:     2,
					Count:     2,
					Limit:     10,
					Offset:    0,
					Table:     "documents",
					Cluster:   "main",
					Operation: "search",
				},
			},
		},
		{
			name: "minimal success response",
			response: &Response{
				Success: true,
				Data:    "operation completed",
			},
		},
		{
			name: "minimal error response",
			response: &Response{
				Success: false,
				Error:   "operation failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize to JSON
			jsonData, err := json.MarshalIndent(tt.response, "", "  ")
			require.NoError(t, err)

			// Deserialize back
			var deserialized Response
			err = json.Unmarshal(jsonData, &deserialized)
			require.NoError(t, err)

			// Verify round-trip consistency
			assert.Equal(t, tt.response.Success, deserialized.Success)
			assert.Equal(t, tt.response.Data, deserialized.Data)
			assert.Equal(t, tt.response.Error, deserialized.Error)
			assert.Equal(t, tt.response.Meta, deserialized.Meta)
		})
	}
}

func TestMeta_JSONSerialization(t *testing.T) {
	meta := &Meta{
		Total:     100,
		Count:     25,
		Limit:     25,
		Offset:    50,
		Table:     "products",
		Cluster:   "ecommerce",
		Operation: "faceted_search",
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, err)

	// Deserialize back
	var deserialized Meta
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, meta.Total, deserialized.Total)
	assert.Equal(t, meta.Count, deserialized.Count)
	assert.Equal(t, meta.Limit, deserialized.Limit)
	assert.Equal(t, meta.Offset, deserialized.Offset)
	assert.Equal(t, meta.Table, deserialized.Table)
	assert.Equal(t, meta.Cluster, deserialized.Cluster)
	assert.Equal(t, meta.Operation, deserialized.Operation)
}
