package httpserver

import (
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	appaccount "wildman-service/internal/app/account"
	appresolution "wildman-service/internal/app/resolution"
	"wildman-service/internal/domain/catalog"
)

type resolutionObservationRequest struct {
	ClientTrackID string    `json:"clientTrackId"`
	FileName      string    `json:"fileName"`
	Title         string    `json:"title"`
	Artists       []string  `json:"artists"`
	Album         string    `json:"album"`
	DiscNumber    *int      `json:"discNumber"`
	TrackNumber   *int      `json:"trackNumber"`
	DurationMS    *int64    `json:"durationMs"`
	Format        string    `json:"format"`
	Fingerprint   *string   `json:"fingerprint"`
	ObservedAt    time.Time `json:"observedAt"`
}

type createResolutionResponse struct {
	RequestID string `json:"requestId"`
	Status    string `json:"status"`
}

type resolutionResponse struct {
	RequestID  string                        `json:"requestId"`
	Status     string                        `json:"status"`
	Candidates []resolutionCandidateResponse `json:"candidates"`
}

type resolutionCandidateResponse struct {
	RecordingID string                      `json:"recordingId"`
	Score       float64                     `json:"score"`
	Title       string                      `json:"title"`
	Artists     []string                    `json:"artists"`
	Release     string                      `json:"release"`
	Source      string                      `json:"source"`
	Sources     []string                    `json:"sources"`
	Evidence    []string                    `json:"evidence"`
	Conflicts   []string                    `json:"conflicts"`
	TagPatch    []catalog.TagPatchOperation `json:"tagPatch"`
}

type resolutionReviewRequest struct {
	Decision           catalog.ReviewDecision  `json:"decision"`
	RecordingID        string                  `json:"recordingId"`
	WritebackStatus    catalog.WritebackStatus `json:"writebackStatus"`
	WritebackErrorCode string                  `json:"writebackErrorCode"`
}

type resolutionReviewResponse struct {
	RequestID          string                  `json:"requestId"`
	Decision           catalog.ReviewDecision  `json:"decision"`
	RecordingID        string                  `json:"recordingId"`
	WritebackStatus    catalog.WritebackStatus `json:"writebackStatus"`
	WritebackErrorCode string                  `json:"writebackErrorCode,omitempty"`
}

func registerResolutionRoutes(router chi.Router, service *appresolution.Service, accountService *appaccount.Service) {
	router.Post("/resolutions", func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := authenticatedClientID(r.Context())
		if !ok {
			writeClientAuthenticationError(w, r, "CLIENT_AUTH_REQUIRED", "缺少或无效客户端 Token")
			return
		}
		if !requestIsJSON(r) {
			writeAPIError(w, r, http.StatusUnsupportedMediaType, "CONTENT_TYPE_UNSUPPORTED", "请求正文必须是 JSON")
			return
		}
		idempotencyKey, err := requestIdempotencyKey(r)
		if err != nil {
			if errors.Is(err, appresolution.ErrIdempotencyKeyRequired) {
				writeAPIError(w, r, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "缺少 Idempotency-Key")
			} else {
				writeAPIError(w, r, http.StatusBadRequest, "IDEMPOTENCY_KEY_INVALID", "Idempotency-Key 格式无效")
			}
			return
		}

		var input resolutionObservationRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}
		fingerprint := ""
		if input.Fingerprint != nil {
			fingerprint = *input.Fingerprint
		}
		observation, err := service.UpsertObservation(r.Context(), clientID, catalog.TrackObservation{
			ClientTrackID: input.ClientTrackID,
			FileName:      input.FileName,
			Title:         input.Title,
			Artists:       input.Artists,
			Album:         input.Album,
			DiscNumber:    input.DiscNumber,
			TrackNumber:   input.TrackNumber,
			DurationMS:    input.DurationMS,
			Format:        input.Format,
			Fingerprint:   fingerprint,
			ObservedAt:    input.ObservedAt,
		})
		if errors.Is(err, catalog.ErrObservationInvalid) {
			writeAPIError(w, r, http.StatusUnprocessableEntity, "OBSERVATION_INVALID", "曲目观测不符合输入限制")
			return
		}
		if err != nil {
			writeInternalAPIError(w, r, err)
			return
		}
		if accountService != nil {
			if err := accountService.ConsumeQuota(r.Context(), clientID, idempotencyKey); err != nil {
				if errors.Is(err, appaccount.ErrQuotaExceeded) {
					writeAPIError(w, r, http.StatusTooManyRequests, "ACCOUNT_QUOTA_EXCEEDED", "账户本月解析额度已用尽")
				} else {
					writeInternalAPIError(w, r, err)
				}
				return
			}
		}

		request, _, err := service.CreateRequest(r.Context(), clientID, observation.ID, idempotencyKey)
		if err != nil {
			writeInternalAPIError(w, r, err)
			return
		}
		writeJSON(w, r, http.StatusAccepted, createResolutionResponse{
			RequestID: request.ID,
			Status:    string(request.Status),
		})
	})

	router.Get("/resolutions/{requestId}", func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := authenticatedClientID(r.Context())
		if !ok {
			writeClientAuthenticationError(w, r, "CLIENT_AUTH_REQUIRED", "缺少或无效客户端 Token")
			return
		}
		result, err := service.GetResult(r.Context(), clientID, chi.URLParam(r, "requestId"))
		switch {
		case err == nil:
			candidates := make([]resolutionCandidateResponse, 0, len(result.Candidates))
			for _, candidate := range result.Candidates {
				artists := make([]string, 0, len(candidate.Recording.Artists))
				for _, artist := range candidate.Recording.Artists {
					artists = append(artists, artist.Name)
				}
				release := ""
				if candidate.Release != nil {
					release = candidate.Release.Title
				}
				candidates = append(candidates, resolutionCandidateResponse{
					RecordingID: candidate.Recording.ID,
					Score:       candidate.Score,
					Title:       candidate.Recording.Title,
					Artists:     artists,
					Release:     release,
					Source:      candidate.Source,
					Sources:     candidate.Sources,
					Evidence:    candidate.Evidence,
					Conflicts:   candidate.Conflicts,
					TagPatch:    candidate.TagPatch,
				})
			}
			writeJSON(w, r, http.StatusOK, resolutionResponse{
				RequestID:  result.Request.ID,
				Status:     string(result.Request.Status),
				Candidates: candidates,
			})
		case errors.Is(err, appresolution.ErrResolutionNotFound):
			writeAPIError(w, r, http.StatusNotFound, "RESOLUTION_NOT_FOUND", "解析请求不存在")
		default:
			writeInternalAPIError(w, r, err)
		}
	})

	router.Post("/resolutions/{requestId}/review", func(w http.ResponseWriter, r *http.Request) {
		clientID, ok := authenticatedClientID(r.Context())
		if !ok {
			writeClientAuthenticationError(w, r, "CLIENT_AUTH_REQUIRED", "缺少或无效客户端 Token")
			return
		}
		if !requestIsJSON(r) {
			writeAPIError(w, r, http.StatusUnsupportedMediaType, "CONTENT_TYPE_UNSUPPORTED", "请求正文必须是 JSON")
			return
		}
		var input resolutionReviewRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeAPIError(w, r, http.StatusBadRequest, "INVALID_REQUEST_BODY", "请求内容不是有效的 JSON")
			return
		}
		review, err := service.ReportReview(r.Context(), clientID, chi.URLParam(r, "requestId"), input.Decision, input.RecordingID, input.WritebackStatus, input.WritebackErrorCode)
		switch {
		case err == nil:
			writeJSON(w, r, http.StatusOK, resolutionReviewResponse{
				RequestID: review.RequestID, Decision: review.Decision, RecordingID: review.RecordingID,
				WritebackStatus: review.WritebackStatus, WritebackErrorCode: review.WritebackErrorCode,
			})
		case errors.Is(err, appresolution.ErrReviewInvalid):
			writeAPIError(w, r, http.StatusUnprocessableEntity, "REVIEW_INVALID", "审核或写回结果不符合限制")
		case errors.Is(err, appresolution.ErrResolutionNotFound):
			writeAPIError(w, r, http.StatusNotFound, "RESOLUTION_NOT_FOUND", "解析请求不存在")
		default:
			writeInternalAPIError(w, r, err)
		}
	})
}

func requestIsJSON(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && strings.EqualFold(mediaType, "application/json")
}

func requestIdempotencyKey(r *http.Request) (string, error) {
	values := r.Header.Values("Idempotency-Key")
	if len(values) == 0 || values[0] == "" {
		return "", appresolution.ErrIdempotencyKeyRequired
	}
	if len(values) != 1 {
		return "", appresolution.ErrIdempotencyKeyInvalid
	}
	value := values[0]
	if len(value) > appresolution.MaxIdempotencyKeyLength || strings.TrimSpace(value) != value {
		return "", appresolution.ErrIdempotencyKeyInvalid
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return "", appresolution.ErrIdempotencyKeyInvalid
		}
	}
	return value, nil
}
