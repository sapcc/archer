// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		panicValue any
		wantStatus int
	}{
		{
			name:       "no panic passes through",
			panicValue: nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "generic panic becomes 500",
			panicValue: fmt.Errorf("boom"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "context canceled becomes 499",
			panicValue: context.Canceled,
			wantStatus: clientClosedRequest,
		},
		{
			name:       "deadline exceeded becomes 499",
			panicValue: context.DeadlineExceeded,
			wantStatus: clientClosedRequest,
		},
		{
			name:       "wrapped context canceled becomes 499",
			panicValue: fmt.Errorf("failed to send query: %w", context.Canceled),
			wantStatus: clientClosedRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.panicValue != nil {
					panic(tt.panicValue)
				}
				w.WriteHeader(http.StatusOK)
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, "/service/123", nil)

			assert.NotPanics(t, func() {
				handler.ServeHTTP(rec, req)
			})
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
