package delete_test

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"testing"
	resp2 "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/delete/mocks"
)

func TestDeleteHandler(t *testing.T) {
	cases := []struct {
		name      string
		uri       string
		respError string
		mockError error
		code      int
	}{
		{
			name: "Success",
			uri:  "/url/10",
			code: http.StatusOK,
		},
		{
			name:      "Invalid ID",
			uri:       "/url/XXX",
			respError: "invalid id",
			code:      http.StatusOK,
		},
		{
			name: "Omitted ID",
			uri:  "/url/",
			respError: "Don't delete it! We do not check this message because it is generated" +
				" by the router. But it must not be empty for the test to work correctly.",
			code: http.StatusNotFound,
		},
		{
			name:      "ID Not Found",
			uri:       "/url/10",
			respError: "url id not found",
			mockError: storage.ErrURLNotFound,
			code:      http.StatusOK,
		},
		{
			name:      "Delete Error",
			uri:       "/url/10",
			respError: "failed to delete url",
			mockError: errors.New("unexpected error"),
			code:      http.StatusOK,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlDeleterMock := mocks.NewURLDeleter(t)

			if tc.respError == "" || tc.mockError != nil {
				urlDeleterMock.On("DeleteURL", mock.AnythingOfType("int64")).
					Return(tc.mockError).
					Once()
			}

			handler := chi.NewRouter()
			handler.Delete("/url/{id}", delete.New(slogdiscard.NewDiscardLogger(), urlDeleterMock))

			req, err := http.NewRequest(http.MethodDelete, tc.uri, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, rr.Code, tc.code)

			if rr.Code == http.StatusOK {
				body := rr.Body.String()

				var resp resp2.Response

				require.NoError(t, json.Unmarshal([]byte(body), &resp))

				require.Equal(t, tc.respError, resp.Error)
			}
		})
	}
}
