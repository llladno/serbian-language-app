package api

import "net/http"

// This file holds the /api/me* handlers. They are 501 stubs wired into the
// router by Task 11; Tasks 12-16 replace each body with the real flow.

func (h handlers) getMe(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) patchMe(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) changePassword(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) linkTelegram(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) unlinkTelegram(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) deleteSession(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) deleteMe(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}
