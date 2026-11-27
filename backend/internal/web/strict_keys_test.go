package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"backend_si/internal/security"
)

var (
	strictInvalidByte = string([]byte{0xff})
	strictTruncated   = string([]byte{0xc3})
	strictOverlong    = string([]byte{0xc0, 0xaf})
	strictSurrogate   = string([]byte{0xed, 0xa0, 0x80})
)

type strictCase struct {
	name string
	body string
}

func strictLoginCases() []strictCase {
	pw := `"senha":"` + flowPassword + `"`
	return []strictCase{
		{"LOGIN", `{"LOGIN":"admin",` + pw + `}`},
		{"Login", `{"Login":"admin",` + pw + `}`},
		{"Senha", `{"login":"admin","Senha":"` + flowPassword + `"}`},
		{"SENHA", `{"login":"admin","SENHA":"` + flowPassword + `"}`},
		{"login twice", `{"login":"nobody",` + pw + `,"login":"admin"}`},
		{"login and LOGIN", `{"login":"nobody",` + pw + `,"LOGIN":"admin"}`},
		{"senha and SENHA", `{"login":"admin","senha":"a wrong long password","SENHA":"` + flowPassword + `"}`},
		{"invalid byte in login", `{"login":"adm` + strictInvalidByte + `in",` + pw + `}`},
		{"invalid byte in senha", `{"login":"admin","senha":"` + flowPassword + strictInvalidByte + `"}`},
		{"truncated character", `{"login":"admin","senha":"` + flowPassword + strictTruncated + `"}`},
		{"overlong encoding", `{"login":"admin` + strictOverlong + `",` + pw + `}`},
		{"encoded surrogate", `{"login":"admin","senha":"` + flowPassword + strictSurrogate + `"}`},
		{"invalid byte in a key", `{"login":"admin",` + pw + `,"` + strictInvalidByte + `":"x"}`},
	}
}

func TestLoginRefusesKeyVariantsDuplicatesAndInvalidUTF8(t *testing.T) {
	t.Parallel()

	for _, tt := range strictLoginCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFlowApp(t)
			anon, token := f.csrf(t, nil)

			rr := f.send(t, http.MethodPost, "/api/login", tt.body, anon,
				map[string]string{security.CSRFHeaderName: token})
			checkBadRequest(t, rr.Code, rr.Body.Bytes())
			if sessionCookie(rr) != nil {
				t.Error("a refused login set a session cookie")
			}
			if n := f.sessions.count(); n != 1 {
				t.Errorf("%d sessions stored, want only the anonymous one", n)
			}
			rr = f.send(t, http.MethodPost, "/api/login", loginBody(flowLogin, flowPassword), anon,
				map[string]string{security.CSRFHeaderName: token})
			if rr.Code != http.StatusOK {
				t.Errorf("exact login after the refusal: status = %d, want 200 (%s)", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestLoginWithMultiByteCharactersStillDecodes(t *testing.T) {
	t.Parallel()

	f := newFlowApp(t)
	anon, token := f.csrf(t, nil)
	rr := f.send(t, http.MethodPost, "/api/login", `{"login":"joão","senha":"Sênha-çã-😀-longa-o-bastante"}`, anon,
		map[string]string{security.CSRFHeaderName: token})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (%s)", rr.Code, rr.Body.String())
	}
}

func strictCadastroCases() []strictCase {
	const login = `"login":"maria.silva"`
	pw := `"senha":"` + cadastroPassword + `"`
	return []strictCase{
		{"Papel", `{` + login + `,` + pw + `,"Papel":"admin"}`},
		{"PAPEL", `{` + login + `,` + pw + `,"PAPEL":"admin"}`},
		{"LOGIN", `{"LOGIN":"maria.silva",` + pw + `,"papel":"comum"}`},
		{"Senha", `{` + login + `,"Senha":"` + cadastroPassword + `","papel":"comum"}`},
		{"papel twice", `{` + login + `,` + pw + `,"papel":"comum","papel":"admin"}`},
		{"papel and PAPEL", `{` + login + `,` + pw + `,"papel":"comum","PAPEL":"admin"}`},
		{"login and Login", `{` + login + `,` + pw + `,"papel":"comum","Login":"joao"}`},
		{"invalid byte in login", `{"login":"maria` + strictInvalidByte + `silva",` + pw + `,"papel":"comum"}`},
		{"invalid byte in senha", `{` + login + `,"senha":"` + cadastroPassword + strictInvalidByte + `","papel":"comum"}`},
		{"truncated character", `{` + login + `,"senha":"` + cadastroPassword + strictTruncated + `","papel":"comum"}`},
		{"overlong encoding", `{` + login + `,"senha":"` + cadastroPassword + strictOverlong + `","papel":"comum"}`},
		{"encoded surrogate", `{` + login + `,"senha":"` + cadastroPassword + strictSurrogate + `","papel":"comum"}`},
		{"invalid byte in a key", `{` + login + `,` + pw + `,"papel":"comum","` + strictInvalidByte + `":"x"}`},
	}
}

func TestCadastroRefusesKeyVariantsDuplicatesAndInvalidUTF8(t *testing.T) {
	t.Parallel()

	for _, tt := range strictCadastroCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var logBuf bytes.Buffer
			repo := newMemUserRepo()
			f := newCadastroApp(t, &logBuf, repo)
			cookie, token := f.loginAs(t, flowLogin)
			csrf := map[string]string{security.CSRFHeaderName: token}
			logBuf.Reset()

			rr := f.send(t, http.MethodPost, "/api/cadastro", tt.body, cookie, csrf)
			checkBadRequest(t, rr.Code, rr.Body.Bytes())
			if n := repo.count(); n != 0 {
				t.Errorf("repository holds %d users, want 0", n)
			}
			if auditLine(logBuf.String()) != "" {
				t.Error("a refused registration was audited")
			}
			if strings.Contains(logBuf.String(), cadastroPassword) {
				t.Errorf("the log has the password:\n%s", logBuf.String())
			}

			rr = f.send(t, http.MethodPost, "/api/cadastro", cadastroBody("maria.silva", cadastroPassword, "comum"), cookie, csrf)
			if rr.Code != http.StatusCreated {
				t.Errorf("exact registration after the refusal: status = %d, want 201 (%s)", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestCadastroWithMultiByteCharactersStillWorks(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	repo := newMemUserRepo()
	f := newCadastroApp(t, &logBuf, repo)
	cookie, token := f.loginAs(t, flowLogin)
	rr := f.send(t, http.MethodPost, "/api/cadastro", cadastroBody("joao.silva", "Sênha-çã-😀-longa-o-bastante", "comum"), cookie,
		map[string]string{security.CSRFHeaderName: token})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rr.Code, rr.Body.String())
	}
	if n := repo.count(); n != 1 {
		t.Errorf("repository holds %d users, want 1", n)
	}
}

func checkBadRequest(t *testing.T, code int, body []byte) {
	t.Helper()
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (%s)", code, body)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("body is not JSON: %v (%s)", err, body)
	}
	if len(got) != 3 || got["error"] != "bad_request" || got["message"] == nil || got["request_id"] == nil {
		t.Errorf("body = %s, want {error: bad_request, message, request_id}", body)
	}
	for _, secret := range []string{flowPassword, cadastroPassword, "maria", "admin"} {
		if bytes.Contains(body, []byte(secret)) {
			t.Errorf("body %s quotes %q from the request", body, secret)
		}
	}
}
