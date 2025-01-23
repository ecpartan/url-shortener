package save

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/slog"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

const aliasLength = 6

type URLSaver interface {
	SaveURL(url, alias string) (int64, error)
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))
		var req Request
		err := render.DecodeJSON(r.Body, &req)

		if errors.Is(err, io.EOF) {
			log.Error("empty request body")
			render.JSON(w, r, resp.Error("Empty request body"))
			return
		}

		if err != nil {
			log.Error("decode json error", sl.Err(err))
			render.JSON(w, r, resp.Error("Invalid request body"))
			return
		}

		log.Info("received request", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validationError := err.(validator.ValidationErrors)
			log.Error("validation error", sl.Err(err))
			render.JSON(w, r, resp.ValidationError(validationError))
			return
		}

		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(aliasLength)
		}

		id, err := urlSaver.SaveURL(req.URL, alias)

		if errors.Is(err, storage.ErrURLExists) {
			log.Info("alias already exists", slog.String("alias", alias))
			render.JSON(w, r, resp.Error("Alias already exists"))
			return
		}

		if err != nil {
			log.Error("save url error", sl.Err(err))
			render.JSON(w, r, resp.Error("Failed to save URL"))
			return
		}

		log.Info("url saved", slog.Int64("id", id), slog.String("alias", alias))

		responseOK(w, r, alias)

	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
