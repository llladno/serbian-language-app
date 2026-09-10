package api

import "net/http"

// This file holds the /api/auth/* handlers. They are 501 stubs wired into the
// router by Task 11; Tasks 12-16 replace each body with the real flow.

func (h handlers) register(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) login(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) logout(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) logoutAll(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) verifyEmail(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) resendVerification(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) forgotPassword(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) resetPassword(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) currentSession(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}

func (h handlers) telegramLogin(w http.ResponseWriter, r *http.Request) {
	fail(w, http.StatusNotImplemented, "not implemented")
}
