package api

import (
	"context"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/danilovaalina/goshrink/internal/model"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type Service interface {
	ShortURL(ctx context.Context, link model.Link) (model.Link, error)
	OriginURL(ctx context.Context, shortCode string, click model.Click) (string, error)
	Analytics(ctx context.Context, shortCode string) (model.Analytics, error)
}

type API struct {
	*echo.Echo
	service Service
}

// New создаёт новый API сервер
func New(service Service) *API {
	a := &API{
		Echo:    echo.New(),
		service: service,
	}

	a.Validator = NewCustomValidator()
	a.HTTPErrorHandler = a.errorHandler

	a.Static("/", "web")

	a.POST("/shorten", a.shorten)
	a.GET("/s/:short_url", a.redirect)
	a.GET("/analytics/:short_url", a.analytics)

	return a
}

type shortenRequest struct {
	URL        string `json:"url" validate:"required,url"`
	CustomCode string `json:"custom_code" validate:"omitempty,shortcode,max=50"`
}

func (a *API) shorten(c echo.Context) error {
	var req shortenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	link, err := a.service.ShortURL(c.Request().Context(), model.Link{
		OriginURL: req.URL,
		ShortCode: req.CustomCode,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, a.linkToResponse(link))
}

func (a *API) redirect(c echo.Context) error {
	shortCode := c.Param("short_url")

	click := model.Click{
		UserAgent: c.Request().UserAgent(),
		IPAddress: c.RealIP(),
	}

	url, err := a.service.OriginURL(c.Request().Context(), shortCode, click)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, url)
}

func (a *API) analytics(c echo.Context) error {
	shortCode := c.Param("short_url")

	stats, err := a.service.Analytics(c.Request().Context(), shortCode)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, a.analyticsToResponse(stats))
}

func (a *API) errorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	var msg interface{} = "internal server error"

	switch {
	case errors.Is(err, model.ErrLinkNotFound):
		code = http.StatusNotFound
		msg = "link not found"

	case errors.Is(err, model.ErrCustomCodeTaken):
		code = http.StatusConflict
		msg = "this custom short code is already in use"

	case errors.Is(err, model.ErrAlreadyExists):
		code = http.StatusConflict
		msg = "record already exists"
	default:
		var he *echo.HTTPError
		if errors.As(err, &he) {
			code = he.Code
			msg = he.Message
		}
	}

	if code >= 500 {
		log.Error().
			Stack().
			Err(err).
			Str("method", c.Request().Method).
			Str("uri", c.Request().RequestURI).
			Msg("internal server error")
	}

	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, echo.Map{"error": msg})
		}
		if err != nil {
			log.Error().Err(err).Msg("failed to send error response")
		}
	}
}

type linkResponse struct {
	ID        string `json:"id"`
	OriginURL string `json:"original_url"`
	ShortCode string `json:"short_code"`
}

type analyticsResponse struct {
	ShortCode   string             `json:"short_code"`
	TotalClicks int                `json:"total_clicks"`
	ByDay       []dailyStatDTO     `json:"by_day"`
	ByUserAgent []userAgentStatDTO `json:"by_user_agent"`
}

type dailyStatDTO struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type userAgentStatDTO struct {
	UserAgent string `json:"user_agent"`
	Count     int    `json:"count"`
}

func (a *API) linkToResponse(l model.Link) linkResponse {
	return linkResponse{
		ID:        l.ID.String(),
		OriginURL: l.OriginURL,
		ShortCode: l.ShortCode,
	}
}

func (a *API) analyticsToResponse(s model.Analytics) analyticsResponse {
	byDay := make([]dailyStatDTO, 0, len(s.ByDay))
	for _, d := range s.ByDay {
		byDay = append(byDay, dailyStatDTO{Date: d.Date, Count: d.Count})
	}

	byUA := make([]userAgentStatDTO, 0, len(s.ByUserAgent))
	for _, u := range s.ByUserAgent {
		byUA = append(byUA, userAgentStatDTO{UserAgent: u.UserAgent, Count: u.Count})
	}

	return analyticsResponse{
		ShortCode:   s.ShortCode,
		TotalClicks: s.TotalClicks,
		ByDay:       byDay,
		ByUserAgent: byUA,
	}
}
