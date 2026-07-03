package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/service"
)

type Handler struct {
	trafficService    *service.TrafficService
	repeaterService   *service.RepeaterService
	diagnosticService *service.DiagnosticService
	scanService       *service.ScanService
	templates         *template.Template
}

func NewHandler(
	trafficService *service.TrafficService,
	repeaterService *service.RepeaterService,
	diagnosticService *service.DiagnosticService,
	scanService *service.ScanService,
	templates *template.Template,
) *Handler {
	return &Handler{
		trafficService:    trafficService,
		repeaterService:   repeaterService,
		diagnosticService: diagnosticService,
		scanService:       scanService,
		templates:         templates,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.index)
	mux.HandleFunc("GET /transactions/{id}", h.detail)
	mux.HandleFunc("GET /api/transactions", h.apiListTransactions)
	mux.HandleFunc("GET /api/transactions/{id}", h.apiGetTransaction)
	mux.HandleFunc("GET /api/transactions/{id}/diagnostics", h.apiDiagnostics)
	mux.HandleFunc("POST /api/transactions/{id}/active-scan", h.apiActiveScan)
	mux.HandleFunc("POST /api/repeater/send", h.apiRepeaterSend)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	transactions, err := h.trafficService.ListTransactions(r.Context(), 200)
	if err != nil {
		http.Error(w, "failed to load transactions", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":        "Traffic History",
		"Transactions": transactions,
		"Selected":     nil,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("render index: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	transactions, err := h.trafficService.ListTransactions(r.Context(), 200)
	if err != nil {
		http.Error(w, "failed to load transactions", http.StatusInternalServerError)
		return
	}

	selected, err := h.trafficService.GetTransaction(r.Context(), id)
	if err != nil {
		http.Error(w, "transaction not found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"Title":        "Traffic Detail",
		"Transactions": transactions,
		"Selected":     selected,
		"Diagnostics":  h.diagnosticService.AnalyzeTransaction(selected),
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("render detail: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) apiListTransactions(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	transactions, err := h.trafficService.ListTransactions(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, transactions)
}

func (h *Handler) apiGetTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	transaction, err := h.trafficService.GetTransaction(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	writeJSON(w, http.StatusOK, transaction)
}

func (h *Handler) apiDiagnostics(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	transaction, err := h.trafficService.GetTransaction(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	writeJSON(w, http.StatusOK, h.diagnosticService.AnalyzeTransaction(transaction))
}

func (h *Handler) apiActiveScan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	transaction, err := h.trafficService.GetTransaction(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	report, err := h.scanService.ScanTransaction(r.Context(), transaction)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) apiRepeaterSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var input model.RepeaterRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	transaction, err := h.repeaterService.Send(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, transaction)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

//
func FormatTime(t model.HTTPTransaction) string {
	return t.CreatedAt.Local().Format("15:04:05")
}
